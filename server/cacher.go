package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

func UseCacher() gin.HandlerFunc {
	cacher := cache.New(2*time.Hour, 4*time.Hour)
	return func(c *gin.Context) {
		c.Set("cacher", cacher)
		c.Next()
	}
}
