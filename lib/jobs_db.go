package lib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"time"
)

var migrationsUp = map[string]func(*Ctx){}
var migrationsDown = map[string]func(*Ctx){}

func RegisterMigration(name string, upFn, downFn func(*Ctx)) string {
	if migrationsUp[name] != nil {
		panic(errors.New("RegisterMigration: Migration already exists: " + name))
	}
	migrationsUp[name] = upFn
	migrationsDown[name] = downFn
	return name
}

var _ = RegisterJob("db", func(c *Ctx, args J) {
	cmd := exec.Command("psql", c.DB.url)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		Log("error", "jobs: db", J{"error": err.Error()})
	}
})

var _ = RegisterJob("db-setup", func(c *Ctx, args J) {
	ExecCmd(`psql -c "create role admin with superuser login password 'admin';" || true`)
	ExecCmd(`psql -c "create database ` + Env("APP_NAME", "") + ` with owner admin;" || true`)
	c.Queue.RunJob("db-migrate-up", J{})
})

var _ = RegisterJob("db-reset", func(c *Ctx, args J) {
	c.DB.Close()
	ExecCmd(`psql -c "drop database ` + Env("APP_NAME", "") + `;" || true`)
	ExecCmd(`psql -c "create database ` + Env("APP_NAME", "") + ` with owner admin;" || true`)
	c.DB.Connect()
	c.Queue.RunJob("db-migrate-up", J{})
})

type migration struct {
	ID      string
	Created time.Time
	Ran     bool
}

func migrationsLoad(c *Ctx) []*migration {
	c.DB.Execute(`CREATE TABLE IF NOT EXISTS app_migrations (id text NOT NULL PRIMARY KEY, created timestamptz NOT NULL)`)
	migrationsNames := []string{}
	for name, _ := range migrationsUp {
		migrationsNames = append(migrationsNames, name)
	}
	sort.Strings(migrationsNames)
	migrationsValues := []*migration{}
	migrationsRan := []*migration{}
	c.DB.All(&migrationsRan, `select * from app_migrations`)
top:
	for _, name := range migrationsNames {
		for _, m := range migrationsRan {
			if m.ID == name {
				m.Ran = true
				migrationsValues = append(migrationsValues, m)
				continue top
			}
		}
		migrationsValues = append(migrationsValues, &migration{ID: name, Created: time.Now()})
	}
	return migrationsValues
}

var _ = RegisterJob("db-migrate", func(c *Ctx, args J) {
	migrations := migrationsLoad(c)
	fmt.Printf("\n\n")
	for _, m := range migrations {
		if m.Ran {
			fmt.Printf("    RAN     %s (%s)\n", m.ID, m.Created.Format("2006-01-02 15:04"))
		} else {
			fmt.Printf("    PENDING %s\n", m.ID)
		}
	}
	fmt.Printf("\n\n")
})

var _ = RegisterJob("db-migrate-up", func(c *Ctx, args J) {
	migrations := migrationsLoad(c)
	fmt.Printf("\n\n")
	for _, m := range migrations {
		if m.Ran {
			fmt.Printf("    OK  %s\n", m.ID)
		} else {
			migrationsUp[m.ID](c)
			c.DB.Execute(`insert into app_migrations (id, created) values ($1, now())`, m.ID)
			fmt.Printf("    RAN %s\n", m.ID)
		}
	}
	fmt.Printf("\n\n")
})

var _ = RegisterJob("db-migrate-down", func(c *Ctx, args J) {
	migrations := migrationsLoad(c)
	fmt.Printf("\n\n")
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		if m.Ran {
			migrationsDown[m.ID](c)
			c.DB.Execute(`delete from app_migrations where id = $1`, m.ID)
			fmt.Printf("    DOWN %s\n", m.ID)
			break
		}
	}
	fmt.Printf("\n\n")
})

var _ = RegisterJob("db-migrate-create", func(c *Ctx, args J) {
	Check(os.MkdirAll("migrations", os.ModePerm))
	migrationID := time.Now().UTC().Format("200601021504")
	if args.Get("name") != "" {
		migrationID += "_" + args.Get("name")
	}
	data := fmt.Sprintf("package migrations\n\nimport \"app/lib\"\n\nvar _ = lib.RegisterMigration(\"%s\", func(c *lib.Ctx) {\n\n}, func(c *lib.Ctx) {\n\n})\n", migrationID)
	ioutil.WriteFile("migrations/"+migrationID+".go", []byte(data), 0644)
	fmt.Printf("\n\n    CREATE %s\n\n\n", "migrations/"+migrationID+".go")
})
