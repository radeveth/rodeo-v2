package controllers

import (
	"app/lib"
	"app/models"
	"net/url"
	"time"
)

func LeaderboardView(c *lib.Ctx) {
	var u *models.LeaderboardUser
	var joinedDiscord bool
	address := c.GetCookie("address")
	if address != "" {
		u = &models.LeaderboardUser{}
		c.DB.FirstWhere(u, "address = $1", address)
		if u.ID == "" {
			u.ID = lib.NewID()
			u.Code = lib.NewRandomID()[0:10]
			u.Address = address
			u.Created = time.Now()
			c.DB.Put(u)
			models.LeaderboardPointCredit(c, u.ID, "Connect", u.Address, 100)
			if refCode := c.Param("r", ""); u.Referrer == "" && refCode != "" {
				referrers := []*models.LeaderboardUser{}
				c.DB.AllWhere(&referrers, "code = $1 limit 1", refCode)
				if len(referrers) > 0 && referrers[0].ID != u.ID {
					u.Referrer = referrers[0].ID
					c.DB.Put(u)
					models.LeaderboardPointCredit(c, referrers[0].ID, "User Referred", u.ID, 100)
				}
			}
		}

		discordPoints := []*models.LeaderboardPoint{}
		c.DB.AllWhere(&discordPoints, "user_id = $1 and reason = 'Discord' limit 1", u.ID)
		joinedDiscord = len(discordPoints) > 0
	}

	total := struct{ Total int64 }{}
	c.DB.First(&total, "select sum(points) total from leaderboards_users")
	tab := c.Param("tab", "leaderboard")
	users := []*models.LeaderboardUser{}
	points := []*models.LeaderboardPoint{}
	referrals := []*models.LeaderboardUser{}
	if tab == "leaderboard" {
		c.DB.All(&users, "select *, coalesce((select case when social_name = '' then address else social_name end from leaderboards_users u2 where u2.id = u1.referrer), '') as referrer from leaderboards_users u1 order by points desc limit 250")
	} else if tab == "points" && address != "" {
		c.DB.AllWhere(&points, "user_id = $1 order by created desc limit 250", u.ID)
	} else if tab == "referrals" && address != "" {
		c.DB.AllWhere(&referrals, "referrer = $1 order by created desc limit 250", u.ID)
	}

	var userArb int64
	if u != nil {
		userArb = 240000 * u.Points / total.Total
	}

	c.Render(200, "leaderboard/leaderboard", lib.J{
		"tab":           tab,
		"user":          u,
		"total":         lib.Bn(total.Total, 0),
		"userArb":       userArb,
		"users":         users,
		"points":        points,
		"referrals":     referrals,
		"joinedDiscord": joinedDiscord,
		"error":         c.Param("error", ""),
	})
}

func LeaderboardInvite(c *lib.Ctx) {
	c.Redirect("/leaderboard/?r=" + c.Param("code", ""))
}

func LeaderboardDiscord(c *lib.Ctx) {
	user := &models.LeaderboardUser{}
	c.DB.MustFirstWhere(user, "address = $1", c.GetCookie("address"))
	discordPoints := []*models.LeaderboardPoint{}
	c.DB.AllWhere(&discordPoints, "user_id = $1 and reason = 'Discord' limit 1", user.ID)
	if len(discordPoints) == 0 {
		models.LeaderboardPointCredit(c, user.ID, "Discord", "", 100)
	}
	c.Redirect(lib.Env("DISCORD_URL", "/"))
}

func LeaderboardX(c *lib.Ctx) {
	base := "https://twitter.com/i/oauth2/authorize"
	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("code_challenge", "challenge")
	query.Set("code_challenge_method", "plain")
	query.Set("scope", "tweet.read users.read follows.read")
	query.Set("state", lib.NewRandomID())
	query.Set("client_id", lib.Env("X_CLIENT_ID", ""))
	query.Set("redirect_uri", lib.Env("BASE_URL", "")+"/leaderboard/x-auth/")
	c.Redirect(base + "?" + query.Encode())
}

func LeaderboardXAuth(c *lib.Ctx) {
	response := struct{ Access_token string }{}
	err := lib.PostFormErr("https://api.twitter.com/2/oauth2/token", &response, map[string]string{
		"Authorization": "Basic " + lib.StringToBase64(lib.Env("X_CLIENT_ID", "")+":"+lib.Env("X_CLIENT_SECRET", "")),
	}, map[string]string{
		"code":          c.Param("code", ""),
		"client_id":     lib.Env("X_CLIENT_ID", ""),
		"code_verifier": "challenge",
		"grant_type":    "authorization_code",
		"redirect_uri":  lib.Env("BASE_URL", "") + "/leaderboard/x-auth/",
	})
	if err != nil {
		c.Redirect("/leaderboard/?error=Error connecting twitter account&e=%v", err)
		return
	}
	token := response.Access_token

	response2 := struct {
		Data struct {
			Id                string
			Name              string
			Username          string
			Profile_image_url string
		}
	}{}
	lib.GetJSONErr("https://api.twitter.com/2/users/me?user.fields=id,name,username,location,profile_image_url,verified", &response2, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/x-www-form-urlencoded",
	})
	if err != nil {
		c.Redirect("/leaderboard/?error=Error fetching profile information&e=%v", err)
		return
	}

	user := &models.LeaderboardUser{}
	c.DB.MustFirstWhere(user, "address = $1", c.GetCookie("address"))
	user.SocialID = response2.Data.Id
	user.SocialName = response2.Data.Name
	user.SocialUsername = response2.Data.Username
	user.SocialPicture = response2.Data.Profile_image_url
	c.DB.Put(user)
	models.LeaderboardPointCredit(c, user.ID, "X", user.SocialID, 100)
	c.Redirect("/leaderboard/")
}
