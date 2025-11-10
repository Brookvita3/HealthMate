package handlers

import (
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func WebSocketProxy(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, _ := url.Parse(target)

		proxy := httputil.NewSingleHostReverseProxy(u)

		c.Request.URL.Scheme = "ws"
		c.Request.URL.Host = u.Host
		c.Request.Host = u.Host

		c.Writer.Header().Set("Connection", "upgrade")
		c.Writer.Header().Set("Upgrade", "websocket")

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
