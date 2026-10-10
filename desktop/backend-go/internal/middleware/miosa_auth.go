package middleware

import "github.com/gin-gonic/gin"

func MiosaAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}
