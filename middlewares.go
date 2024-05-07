package main

import (
	"app/lib"
	"app/models"
	"time"
)

var rdoPrice *lib.BigInt

func midSession(c *lib.Ctx) {
	var session *models.Session
	c.Data["session"] = session

	sessionID := c.GetCookie(lib.SessionCookieName)
	if sessionID != "" {
		c.DB.FirstWhere(session, "id = $1", sessionID)
		if session.ID != "" {
			// Check expiry
			if session.Expires.Before(time.Now().UTC()) {
				c.DB.Delete(session)
				session = nil
				c.Data["session"] = session
			}

			// Refresh expiry if < 12 days left
			if session.Expires.Sub(time.Now().UTC()).Hours()/24 <= 12 {
				session.Expires = time.Now().UTC().Add(14 * 24 * time.Hour)
				c.DB.Put(session)
			}
		} else {
			c.SetCookie(lib.SessionCookieName, "")
		}
	}

	if session != nil {
		user := &models.User{}
		c.DB.MustFirstWhere(user, "id = $1", session.UserID)
		c.Data["currentUser"] = user
	}

	c.Data["address"] = c.GetCookie("address")
	if rdoPrice == nil {
		client := c.Server.ChainClients[models.DefaultChainId]
		rdoPrice = client.CallUint("0x309349d5D02C6f8b50b5040e9128E1A8375042D7", "latestAnswer--int256")
	}
	c.Data["priceRdo"] = rdoPrice
}

func midAuth(c *lib.Ctx) {
	session := c.Data["session"]
	if session != nil {
		c.Redirect("/signin/?return=" + c.Req.URL.Path)
		return
	}
}
