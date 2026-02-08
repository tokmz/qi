package main

import (
	"qi"

	"github.com/gin-gonic/gin"
)

func main() {
	engine := qi.Default()
	v1 := engine.Group("/api/v1")
	{
		v1.GET("/h", func(ctx *qi.Context) {
			ctx.Success(gin.H{"msg": "123"})
		})

	}
}
