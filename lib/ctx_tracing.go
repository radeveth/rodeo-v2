package lib

import "time"

type Span struct {
	ID        string
	TraceID   string
	ParentID  string
	Timestamp int64
	Duration  int64
	Name      string
	Tags      J
	Service   string
}

var spansService = Env("APP_NAME", "")
var spansPending = []*Span{}

func (c *Ctx) TraceEvent(name string, tags J) {
	spansPending = append(spansPending, &Span{
		ID:        NewID(),
		TraceID:   c.tracingTraceID,
		ParentID:  c.tracingSpanID,
		Timestamp: time.Now().UnixNano() / 1000,
		Duration:  0,
		Name:      name,
		Tags:      tags,
		Service:   spansService,
	})
}

func (c *Ctx) TraceSpan(name string, tags J, start time.Time, duration int64) {
	spansPending = append(spansPending, &Span{
		ID:        NewID(),
		TraceID:   c.tracingTraceID,
		ParentID:  c.tracingSpanID,
		Timestamp: start.UnixNano() / 1000,
		Duration:  duration,
		Name:      name,
		Tags:      tags,
		Service:   spansService,
	})
}

func (c *Ctx) TraceSpanFn(name string, tags J, fn func()) {
	start := time.Now().UnixNano() / 1000
	fn()
	spansPending = append(spansPending, &Span{
		ID:        NewID(),
		TraceID:   c.tracingTraceID,
		ParentID:  c.tracingSpanID,
		Timestamp: start,
		Duration:  (time.Now().UnixNano() / 1000) - start,
		Name:      name,
		Tags:      tags,
		Service:   spansService,
	})
}

func (c *Ctx) TraceSet(key string, value interface{}) {
	c.tracingRootTags[key] = value
}

func (c *Ctx) TraceSpanRoot(name string, tags J, start time.Time, duration int64) {
	spansPending = append(spansPending, &Span{
		ID:        c.tracingSpanID,
		TraceID:   c.tracingTraceID,
		ParentID:  "",
		Timestamp: start.UnixNano() / 1000,
		Duration:  duration,
		Name:      name,
		Tags:      c.tracingRootTags,
		Service:   spansService,
	})
}
