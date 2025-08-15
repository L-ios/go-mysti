package controllers

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

type IndexController struct {
	db *sql.DB
}

func RegisterRoutes(db *sql.DB, router *gin.RouterGroup) IndexController {
	ctl := IndexController{db: db}
	router.GET("/", ctl.index)

	return ctl
}

func (ctl *IndexController) index(ctx *gin.Context) {
	ctx.HTML(200, "posts/edit/index.html", gin.H{
		"title": "Main website",
	})
}
