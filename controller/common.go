package controller

import (
	"net/http"

	"ivorareteambot/types"

	"github.com/gin-gonic/gin"
)

func (c Controller) respondMessage(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusOK, types.NewMessage(message))
}
