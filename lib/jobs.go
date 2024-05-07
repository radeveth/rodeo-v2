package lib

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"time"
)

var _ = RegisterJob("help", func(c *Ctx, args J) {
	jobNames := []string{}
	for k, _ := range jobs {
		jobNames = append(jobNames, k)
	}
	sort.Strings(jobNames)
	fmt.Printf("\n")
	for _, name := range jobNames {
		fmt.Printf("  %s\n", name)
	}
	fmt.Printf("\n")
})

var _ = RegisterJob("start", func(c *Ctx, args J) {
	c.Server.Scheduler.Start()
	c.Server.Queue.Start()
	c.Server.Start()
})

var _ = RegisterSchedule("cleanup", time.Hour)

var _ = RegisterJob("cleanup", func(c *Ctx, args J) {
	c.DB.Execute("delete from app_cache where expires < now()")
})

var _ = RegisterJob("cache-clear", func(c *Ctx, args J) {
	c.DB.Execute("truncate table app_cache")
})

var _ = RegisterJob("generate-secret", func(c *Ctx, args J) {
	random := make([]byte, 32)
	_, err := crand.Read(random)
	Check(err)
	secret := hex.EncodeToString(random)
	fmt.Printf("\n%s\n\n", secret)
})
