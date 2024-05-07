package models

import (
	"app/lib"
	"time"
)

type Session struct {
	ID      string    `json:"id"`
	UserID  string    `json:"userId"`
	Data    lib.J     `json:"data"`
	Expires time.Time `json:"expires"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}
