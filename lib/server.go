package lib

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

// Server represents an HTTP server, registered routes, and it's associated services
type Server struct {
	FS            embed.FS
	routes        []*Route
	middlewares   []HandlerFunc
	notFound      HandlerFunc
	assetsHandler http.Handler
	Tpl           *template.Template
	Database      *Database
	Cache         *Cache
	Storage       *Storage
	Queue         *JobQueue
	Scheduler     *Scheduler
	ChainClients  map[int64]*ChainClient
}

// Route represents a route the HTTP server can handler (we compile the user provided path into a regexp)
type Route struct {
	path string
	re   *regexp.Regexp
	fns  []HandlerFunc
}

// HandlerFunc represents a route handler the HTTP server can dispatch requests to
type HandlerFunc func(c *Ctx)

// NewServer creates a new server instance
func NewServer(fs embed.FS) *Server {
	s := &Server{}
	s.FS = fs
	s.Tpl = NewTemplateFromFS(fs)
	s.Database = NewDatabase(Env("DATABASE_URL", "postgres://admin:admin@localhost:5432/"+Env("APP_NAME", "app")+"?sslmode=disable"))
	s.Cache = NewCache(s)
	s.Queue = NewJobQueue(s)
	s.Storage = NewStorage(Env("S3_BUCKET", ""), false)
	s.Scheduler = NewScheduler(s)
	s.ChainClients = map[int64]*ChainClient{}

	//isMigrating := len(os.Args) > 0 && (os.Args[1] == "db-migrate" || os.Args[1] == "db-reset")
	isMigrating := false
	if !isMigrating {
		s.Database.Execute(`CREATE TABLE IF NOT EXISTS app_jobs (id text NOT NULL PRIMARY KEY, name text NOT NULL, args jsonb NOT NULL, priority int, created timestamptz NOT NULL)`)
		s.Database.Execute(`CREATE TABLE IF NOT EXISTS app_schedules (id text NOT NULL PRIMARY KEY, last_ran timestamptz NOT NULL, next_run timestamptz NOT NULL)`)
		s.Database.Execute(`CREATE UNLOGGED TABLE IF NOT EXISTS app_cache (id text NOT NULL PRIMARY KEY, value bytea NOT NULL, expires timestamptz NOT NULL)`)
	}

	s.assetsHandler = http.FileServer(http.FS(fs))
	s.Handle("/admin/run-job/", handleAdminRunJob)
	s.Handle("/admin/sign-in-as/", handleAdminSignInAs)
	return s
}

// Middleware adds a new middleware to run on every request. If any of them sends a response, not other handlers will be called.
func (s *Server) Middleware(fn HandlerFunc) {
	s.middlewares = append(s.middlewares, fn)
}

// Handle adds a new route to the HTTP server for a given path. You can provide more than one handler, they will all be ran in sequence.
func (s *Server) Handle(path string, fns ...HandlerFunc) {
	pathRegexp := regexp.MustCompile("/:([a-zA-Z0-9]+)").ReplaceAllStringFunc(path, func(s string) string {
		return "/(?P<" + s[2:] + ">[^/]+)"
	})
	s.routes = append(s.routes, &Route{
		path: path,
		re:   regexp.MustCompile("^" + pathRegexp + "$"),
		fns:  fns,
	})
}

// HandleNotFound sets the handler to use when no other route matches
func (s *Server) HandleNotFound(fn HandlerFunc) {
	s.notFound = fn
}

// Start starts the HTTP server. Serves static assets from `/assets/` too.
func (s *Server) Start() {
	port := Env("PORT", "8000")
	Log("info", "server starting", J{"port": port})
	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: time.Second * 60,
		ReadTimeout:  time.Second * 60,
		IdleTimeout:  time.Second * 60,
		Handler:      http.HandlerFunc(s.handler),
	}
	go func() {
		log.Fatal(server.ListenAndServe())
	}()
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	err := server.Close()
	if err != nil {
		log.Println(err)
	}
	s.Queue.Stop()
	s.Scheduler.Stop()
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		s.assetsHandler.ServeHTTP(w, r)
		return
	}

	c := NewCtx(s)
	c.Req = r
	c.Res = w

	start := time.Now()
	defer func() {
		Log("info", "request", J{"method": r.Method, "path": r.URL.Path, "time": time.Now().Sub(start).Milliseconds()})
	}()

	// Handle panics
	defer func() {
		if err := recover(); err != nil {
			stackLines := strings.Split(string(debug.Stack()), "\n")
			errorLine := ""
			for _, v := range stackLines {
				if strings.Contains(v, "app/handlers/") {
					parts := strings.Split(v, "/")
					errorLine = parts[len(parts)-1]
					break
				}
			}
			if len(stackLines) > 8 {
				stackLines = stackLines[7:]
			}
			Log("error", "request panic", J{
				"error":     fmt.Sprintf("%v", err),
				"errorLine": errorLine,
				"stack":     stackLines,
			})
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(500)
					if Env("ENV", "") == "dev" {
						w.Write([]byte(fmt.Sprintf("%v\n%s\n\n%s", err, errorLine, strings.Join(stackLines, "\n"))))
					} else {
						w.Write([]byte("Server error"))
					}
				}
			}()
			c.Render(500, "other/500", J{
				"errorName":  err,
				"errorLine":  errorLine,
				"stack":      strings.Join(stackLines, "\n"),
				"paramsUrl":  c.Req.URL.Query(),
				"paramsForm": c.Req.Form,
			})
		}
	}()

	// Loop handlers and match path patterns
	for _, ro := range s.routes {
		match := ro.re.FindStringSubmatch(r.URL.Path)
		if len(match) == 0 {
			continue
		}
		// Set path params in query params
		for i, v := range match {
			if ro.re.SubexpNames()[i] != "" {
				c.params.Set(ro.re.SubexpNames()[i], v)
			}
		}

		// Loop middlewares
		for _, m := range s.middlewares {
			m(c)
			if c.Code != 0 {
				return
			}
		}
		// Create request context and run middleware/handler list till we send code
		for _, h := range ro.fns {
			h(c)
			if c.Code != 0 {
				return
			}
		}
		return
	}

	// No route matched, show not found
	for _, m := range s.middlewares {
		m(c)
		if c.Code != 0 {
			return
		}
	}
	if s.notFound != nil {
		s.notFound(c)
	} else {
		w.WriteHeader(404)
		w.Write([]byte("Page not found"))
	}
}
