package controllers

import (
	"app/lib"
	"app/models"
	"time"
)

func MarketingHome(c *lib.Ctx) {
	strategies := models.Strategies
	c.Cache.Try("strategies", &strategies, 5*time.Minute, cacheStrategies(c))
	c.Render(200, "marketing/home", lib.J{"strategies": strategies})
}

func MarketingPostsList(c *lib.Ctx) {
	posts := []*models.Post{}
	c.DB.AllWhere(&posts, "deleted is null and published order by created desc")
	c.Render(200, "marketing/posts-list", lib.J{
		"title": "Blog",
		"posts": posts,
	})
}

func MarketingPostsView(c *lib.Ctx) {
	post := &models.Post{}
	c.DB.FirstWhere(post, "slug = $1", c.Param("slug", ""))
	if post.ID == "" {
		c.Render(404, "other/404", lib.J{})
		return
	}
	c.Render(200, "marketing/posts-view", lib.J{
		"title": post.Title,
		"post":  post,
	})
}

func MarketingDocsView(c *lib.Ctx) {
	doc := &models.Doc{}
	c.DB.FirstWhere(doc, "slug = $1", c.Param("slug", ""))
	if doc.ID == "" {
		c.Render(404, "other/404", lib.J{})
		return
	}
	c.Render(200, "marketing/docs-view", lib.J{
		"title": doc.Title,
		"doc":   doc,
	})
}
