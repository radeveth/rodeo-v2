package controllers

import (
	"app/lib"
	"app/models"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const BCryptCost int = 12

func AuthSignin(c *lib.Ctx) {
	session := c.Data["session"].(*models.Session)
	if session != nil {
		c.Redirect("/")
		return
	}

	errors := []string{}
	if c.Req.Method == "POST" {
		user := &models.User{}
		c.DB.FirstWhere(user, "email = $1", c.Param("email", ""))
		if user.ID == "" {
			errors = append(errors, "No user for that email")
			goto render
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(c.Param("password", ""))); err != nil {
			errors = append(errors, "Wrong password")
			goto render
		}

		session := &models.Session{
			ID:      lib.NewID(),
			UserID:  user.ID,
			Data:    lib.J{},
			Expires: time.Now().UTC().Add(14 * 24 * time.Hour),
			Created: time.Now().UTC(),
			Updated: time.Now().UTC(),
		}
		c.DB.Put(session)
		c.SetCookie(lib.SessionCookieName, session.ID)
		c.Redirect(c.Param("return", "/"))
		return
	}

render:
	c.Render(200, "auth/signin", lib.J{
		"title":  "Log in",
		"email":  c.Param("email", ""),
		"errors": errors,
	})
}

func AuthSignup(c *lib.Ctx) {
	errors := []string{}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Param("password", "")), BCryptCost)
	lib.Check(err)
	user := &models.User{
		ID:       lib.NewID(),
		Name:     c.Param("name", ""),
		Email:    c.Param("email", ""),
		Password: string(hashedPassword),
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}

	if c.Req.Method == "POST" {
		errors = lib.Validate(
			c.Params(),
			lib.ValidatePresence("name"),
			lib.ValidateUnique("email", c.DB, "users", "email", ""),
			lib.ValidateLength("password", 8, -1),
		)
		if len(errors) > 0 {
			goto render
		}

		session := &models.Session{
			ID:      lib.NewID(),
			UserID:  user.ID,
			Data:    lib.J{},
			Expires: time.Now().UTC().Add(14 * 24 * time.Hour),
			Created: time.Now().UTC(),
			Updated: time.Now().UTC(),
		}
		c.DB.Put(user)
		c.DB.Put(session)

		c.SetCookie(lib.SessionCookieName, session.ID)
		c.Redirect("/")
		return
	}

render:
	c.Render(200, "auth/signup", lib.J{
		"title":  "Sign up",
		"user":   user,
		"errors": errors,
	})
}

func AuthForgot(c *lib.Ctx) {
	session := c.Data["session"].(*models.Session)
	if session != nil {
		c.Redirect("/")
		return
	}

	success := false
	errors := []string{}
	if c.Req.Method == "POST" {
		users := []*models.User{}
		c.DB.AllWhere(&users, "email = $1", c.Param("email", ""))
		if len(users) == 0 {
			errors = append(errors, "No user with that email address")
			goto render
		}
		token := lib.CreateToken("reset_"+users[0].ID, lib.Env("SECRET", "keyboardcat"), 6*60)
		subject := lib.Env("COMPANY_NAME", "") + " Password Reset"
		text := `Don't worry we all forget sometimes

You've recently asked to reset the password for this Cortina account: {{.to}}

To update your password, click the button below

If you didn't make the request, you can ignore
this email and do nothing. Another user likely entered your email
address by mistake while trying to reset a password.`
		link := lib.Env("BASE_URL", "http://localhost:"+lib.Env("PORT", "8000")) + "/reset/?token=" + token
		c.SendEmail(users[0].Email, subject, text, "Change password", link)
		success = true
	}

render:
	c.Render(200, "auth/forgot", lib.J{
		"title":   "Forgotten Password",
		"success": success,
		"errors":  errors,
	})
}

func AuthReset(c *lib.Ctx) {
	errors := []string{}
	value, valid := lib.ValidateToken(c.Param("token", ""), lib.Env("SECRET", "keyboardcat"))
	parts := strings.Split(value, "_")
	if !valid || len(parts) != 2 || parts[0] != "reset" {
		errors = append(errors, "Invalid password reset token provided")
		goto render
	}

	if c.Req.Method == "POST" {
		user := &models.User{}
		c.DB.MustFirstWhere(user, "id = $1", parts[1])
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Param("password", "")), BCryptCost)
		lib.Check(err)
		user.Password = string(hashedPassword)
		c.DB.Put(user)
		c.Redirect("/signin/")
		return
	}

render:
	c.Render(200, "auth/reset", lib.J{
		"title":  "Password Reset",
		"errors": errors,
	})
}

func AuthSignout(c *lib.Ctx) {
	session := c.Data["session"].(*models.Session)
	if session != nil {
		c.DB.Delete(session)
	}
	c.SetCookie(lib.SessionCookieName, "")
	c.Redirect("/signin/")
}

func AuthProfile(c *lib.Ctx) {
	user := c.Data["currentUser"].(*models.User)

	if c.Req.Method == "POST" {
		user.Name = c.Param("name", "")
		user.Name = c.Param("username", "")
		if p := c.Param("password", ""); p != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Param("password", "")), BCryptCost)
			lib.Check(err)
			user.Password = string(hashedPassword)
		}
		user.Updated = time.Now().UTC()
		c.DB.Put(user)
		c.Redirect("/")
		return
	}

	c.Render(200, "auth/profile", lib.J{
		"title": "Profile",
		"user":  user,
	})
}
