package models

import (
	"app/lib"
	"time"
)

type LeaderboardUser struct {
	ID             string
	Code           string
	Referrer       string
	Address        string
	SocialID       string
	SocialName     string
	SocialUsername string
	SocialPicture  string
	Points         int64
	PointsReferral int64
	Created        time.Time
}

type LeaderboardPoint struct {
	ID       string
	UserID   string
	Reason   string
	ReasonID string
	Points   int64
	Created  time.Time
}

func LeaderboardPointCredit(c *lib.Ctx, id, reason, reasonID string, points int64) {
	pointID := lib.NewID()
	for id != "" {
		if points == 0 {
			break
		}
		u := &LeaderboardUser{}
		c.DB.MustFirstWhere(u, "id = $1", id)
		c.DB.Put(&LeaderboardPoint{
			ID:       lib.NewID(),
			UserID:   id,
			Reason:   reason,
			ReasonID: reasonID,
			Points:   points,
			Created:  time.Now(),
		})
		if reason == "referral" {
			c.DB.Execute("update leaderboards_users set points = points + $2 where id = $1", id, points)
		} else {
			c.DB.Execute("update leaderboards_users set points = points + $2, points_referral = points_referral + $2 where id = $1", id, points)
		}
		reason = "referral"
		reasonID = pointID
		id = u.Referrer
		points = points / 4
	}
}
