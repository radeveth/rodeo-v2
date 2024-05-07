package lib

import (
	"errors"
	"sync"
	"time"
)

var globalSchedules = map[string]time.Duration{}

func RegisterSchedule(jobName string, runEvery time.Duration) string {
	if globalSchedules[jobName] != 0 {
		panic(errors.New("RegisterSchedule: " + jobName + " already registered"))
	}
	globalSchedules[jobName] = runEvery
	return jobName
}

type Schedule struct {
	ID      string
	LastRan time.Time
	NextRun time.Time
}

type Scheduler struct {
	db    *Database
	queue *JobQueue
	stop  bool
	wg    sync.WaitGroup
}

func NewScheduler(s *Server) *Scheduler {
	return &Scheduler{db: s.Database, queue: s.Queue}
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.start()
}

func (s *Scheduler) start() {
	defer s.wg.Done()

	schedules := []*Schedule{}
	schedulesExisting := map[string]*Schedule{}
	s.db.All(&schedules, "select * from app_schedules")
	// Delete old schedules
	for _, v := range schedules {
		if globalSchedules[v.ID] == 0 {
			s.db.Execute(`delete from app_schedules where id = $1`, v.ID)
			continue
		}
		schedulesExisting[v.ID] = v
	}
	// Create new schedules
	for id, interval := range globalSchedules {
		if schedulesExisting[id] != nil {
			continue
		}
		s.db.Execute(`insert into app_schedules (id, last_ran, next_run) values ($1, $2, $3)`,
			id, time.Now(), time.Now().Add(interval).Round(interval))
	}

	for !s.stop {
		schedules := []*Schedule{}
		s.db.All(&schedules, "select * from app_schedules where next_run < $1", time.Now())
		for _, v := range schedules {
			// Try and update schedule next run first
			interval := globalSchedules[v.ID]
			schedulesUpdated := []*Schedule{}
			s.db.All(&schedulesUpdated, `update app_schedules set last_ran = $3, next_run = $4 where id = $1 and next_run < $2 returning *`, v.ID, time.Now(), time.Now(), time.Now().Add(interval).Round(interval))
			// We were the node to bump the next_run, execute job
			if len(schedulesUpdated) > 0 {
				s.queue.Enqueue(v.ID, J{}, JobPriorityHigh)
			}
		}
		time.Sleep(time.Minute)
	}
}

func (s *Scheduler) Stop() {
	s.stop = true
	s.wg.Wait()
}
