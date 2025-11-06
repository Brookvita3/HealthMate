package handlers

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func ReverseProxy(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		url, err := url.Parse(target)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid target"})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(url)

		// Rewrite path
		c.Request.URL.Path = c.Param("proxyPath")
		c.Request.Host = url.Host
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
