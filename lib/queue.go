package lib

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var jobs = map[string]func(*Ctx, J){}

func RegisterJob(name string, handler func(*Ctx, J)) string {
	if jobs[name] != nil {
		panic(errors.New("RegisterJob: Job already exists: " + name))
	}
	jobs[name] = handler
	return name
}

const (
	JobPriorityHigh   int64 = 1
	JobPriorityMedium int64 = 2
	JobPriorityLow    int64 = 3
)

type Job struct {
	ID       string
	Name     string
	Args     J
	Priority int64
	Created  time.Time
}

type JobQueue struct {
	ctx    *Ctx
	db     *Database
	server *Server
	stop   bool
	wg     sync.WaitGroup
}

func (q *JobQueue) WithCtx(ctx *Ctx) *JobQueue {
	return &JobQueue{db: q.db.WithCtx(ctx), server: q.server, ctx: ctx}
}

func NewJobQueue(s *Server) *JobQueue {
	return &JobQueue{db: s.Database, server: s}
}

func (q *JobQueue) Start() {
	q.wg.Add(1)
	go q.start()
}

func (q *JobQueue) start() {
	defer q.wg.Done()
	for !q.stop {
		jobs := []*Job{}
		q.db.All(&jobs, `delete from app_jobs where id in (select id from app_jobs where created < $1 order by priority, created asc limit 5) returning *`, time.Now())
		for _, job := range jobs {
			q.RunJob(job.Name, job.Args)
		}
		// Sleep if we had nothing to do, else, run the next batch as fast as possible
		if len(jobs) == 0 {
			time.Sleep(time.Second)
		}
	}
}

func (q *JobQueue) Stop() {
	q.stop = true
	q.wg.Wait()
}

func (q *JobQueue) Enqueue(name string, args J, priorityArgs ...int64) {
	// TODO batch inserts
	priority := JobPriorityLow
	if len(priorityArgs) > 0 {
		priority = priorityArgs[0]
	}
	q.db.Execute(`insert into app_jobs (id, name, args, priority, created) values ($1, $2, $3, $4, $5)`,
		NewID(), name, args, priority, time.Now())
}

func (q *JobQueue) Delay(name string, args J, delay time.Duration, priorityArgs ...int64) {
	// TODO batch inserts
	priority := JobPriorityLow
	if len(priorityArgs) > 0 {
		priority = priorityArgs[0]
	}
	q.db.Execute(`insert into app_jobs (id, name, args, priority, created) values ($1, $2, $3, $4, $5)`,
		NewID(), name, args, priority, time.Now().Add(delay))
}

func (q *JobQueue) RunJob(name string, args J) {
	handler, ok := jobs[name]
	if !ok {
		Log("error", "RunJob: No job for given name", J{"name": name})
		return
	}
	c := q.ctx
	if c == nil {
		c = NewCtx(q.server)
	}
	defer func() {
		if err := recover(); err != nil {
			Log("error", "JobQueue: RunJob: panic", J{
				"name":  name,
				"args":  args,
				"error": fmt.Sprintf("%v", err),
				"stack": string(debug.Stack()),
			})
		}
	}()
	handler(c, args)
}

func (q *JobQueue) RunCliJob() {
	jobName := ""
	args := J{}
	if len(os.Args) > 1 {
		jobName = os.Args[1]
	}
	if jobName == "" {
		jobName = "start"
		if Env("ENV", "development") == "dev" {
			jobName = "help"
		}
	}
	for i := 2; i < len(os.Args); i++ {
		parts := strings.SplitN(os.Args[i], "=", 2)
		if len(parts) == 1 {
			args.Set("arg", parts[0])
		} else {
			args.Set(parts[0], parts[1])
		}
	}
	q.RunJob(jobName, args)
}
