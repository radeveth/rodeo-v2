package lib

import (
	"time"
)

func handleAdminRunJob(c *Ctx) {
	if c.Param("secret", "") != Env("ADMIN_SECRET", NewID()) {
		c.Text(403, "Missing valid admin secret")
		return
	}
	c.Queue.RunJob(c.Param("name", ""), J{})
	c.Text(200, c.tracingTraceID)
}

func handleAdminSignInAs(c *Ctx) {
	if c.Param("secret", "") != Env("ADMIN_SECRET", NewID()) {
		c.Text(403, "Missing valid admin secret")
		return
	}
	sessionID := NewID()
	c.DB.Execute(`insert into sessions (id, user_id, data, expires) values ($1, $2, '{}', $3)`,
		sessionID, c.Param("user_id", ""), time.Now().UTC().Add(14*24*time.Hour))
	c.SetCookie(SessionCookieName, sessionID)
	c.Redirect(SessionSigninRedirect)
}
