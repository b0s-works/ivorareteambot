package controller

import (
	"github.com/gin-gonic/gin"
	"ivorareteambot/app"
	"go/test/fixedbugs/issue20682.dir"
	"net/http"
)

const slackTokenKey = "token"

type Controller struct {
	app app.Application
	gin *gin.Engine
	slackToken string
}

func New(app app.Application, slackToken string) Controller {
	return Controller{app: app, gin: gin.Default(), slackToken: slackToken}
}

func (c Controller) Serve() {
	http.Handle("/", c.gin)
}

func (c Controller) InitRouters() {
	slack := c.gin.Group("")
	slack.Use(c.slackAuth)
	slack.POST("/", )
}

func (c Controller) slackAuth(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		//TODO respond valid to Slack error
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		ctx.Abort()
		return
	}

	if t := form.Value[slackTokenKey]; len(t) == 0 || t[0] != c.slackToken {
		//TODO respond error
		ctx.Abort()
		return
	}

	ctx.Next()
}
