package main

import (
	. "app/controllers"
	"app/lib"
)

func setupRoutes(s *lib.Server) {
	s.Tpl = lib.NewTemplateFromFS(FS)
	s.Storage = lib.NewStorage(lib.Env("AWS_BUCKET", ""), false)
	s.Database = lib.NewDatabase(lib.Env("DATABASE_URL", "postgres://admin:admin@localhost:5432/"+lib.Env("APP_NAME", "app")+"?sslmode=disable"))
	s.Middleware(midSession)

	s.Handle("/", MarketingHome)
	s.Handle("/blog/", MarketingPostsList)
	s.Handle("/blog/:slug/", MarketingPostsView)
	s.Handle("/docs/:slug/", MarketingDocsView)
	s.Handle("/leaderboard/", LeaderboardView)
	s.Handle("/leaderboard/x/", LeaderboardX)
	s.Handle("/leaderboard/x-auth/", LeaderboardXAuth)
	s.Handle("/leaderboard/discord/", LeaderboardDiscord)
	s.Handle("/i/:code", LeaderboardInvite)
	s.Handle("/farm/", AppStrategies)
	s.Handle("/farm/:slug/", AppStrategy)
	s.Handle("/earn/", AppLend)
	s.Handle("/silos/", AppStaking)
	s.Handle("/rewards/", AppRewards)
	s.Handle("/vesting/", AppVesting)
	s.Handle("/analytics/", AppAnalytics)

	s.HandleNotFound(func(c *lib.Ctx) {
		c.Render(200, "other/404", lib.J{})
	})
}
