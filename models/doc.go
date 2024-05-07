package models

import (
	"database/sql"
	"time"
)

type Doc struct {
	ID        string       `json:"id"`
	Slug      string       `json:"slug"`
	Title     string       `json:"title"`
	Content   string       `json:"content"`
	Published bool         `json:"published"`
	Created   time.Time    `json:"created"`
	Updated   time.Time    `json:"updated"`
	Deleted   sql.NullTime `json:"deleted"`
}
