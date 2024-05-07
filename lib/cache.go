package lib

import (
	"encoding/json"
	"time"
)

type cacheEntry struct {
	ID      string
	Value   []byte
	Expires time.Time
}

// Cache represents an Database backed cache
type Cache struct {
	ctx *Ctx
	db  *Database
}

// NewCache creates a new Cache
func NewCache(server *Server) *Cache {
	return &Cache{db: server.Database}
}

func (c *Cache) WithCtx(ctx *Ctx) *Cache {
	return &Cache{ctx: ctx, db: c.db.WithCtx(ctx)}
}

func (c *Cache) Get(key string, result interface{}) bool {
	entry := &cacheEntry{}
	err := c.db.FirstErr(entry, `select * from app_cache where id = $1 and expires > now()`, key)
	if err == ErrDatabaseNotFound {
		return false
	}
	Check(err)
	Check(json.Unmarshal(entry.Value, result))
	return true
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	bytes, err := json.Marshal(value)
	Check(err)
	sql := `insert into app_cache (id, value, expires) values ($1, $2, $3)
		on conflict (id) do update set value = excluded.value, expires = excluded.expires`
	c.db.Execute(sql, key, bytes, time.Now().UTC().Add(ttl))
}

func (c *Cache) Delete(key string) {
	c.db.Execute(`delete from app_cache where id = $1`, key)
}

func (c *Cache) Try(key string, result interface{}, ttl time.Duration, fn func() interface{}) {
	if !c.Get(key, result) {
		value := fn()
		c.Set(key, value, ttl)
		bytes, err := json.Marshal(value)
		Check(err)
		Check(json.Unmarshal(bytes, result))
	}
}
