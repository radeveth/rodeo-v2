package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Email    string       `json:"email"`
	Password string       `json:"password"`
	Created  time.Time    `json:"created"`
	Updated  time.Time    `json:"updated"`
	Deleted  sql.NullTime `json:"deleted"`
}
