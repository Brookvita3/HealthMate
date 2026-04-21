package handlers

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func ReverseProxy(target string, basePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		t, err := url.Parse(target)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid target"})
			return
		}

		proxyPath := c.Param("proxyPath")
		if proxyPath == "/" {
			proxyPath = ""
		}
		outbound := basePath + proxyPath

		proxy := &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				r.SetURL(t)
				r.Out.URL.Path = outbound
				r.Out.URL.RawPath = ""
			},
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// ReverseProxyExact dùng cho route không có *proxyPath (vd. GET /groups, GET /medications).
// Tránh dựa vào Param rỗng và join path của NewSingleHostReverseProxy.
func ReverseProxyExact(target string, outboundPath string) gin.HandlerFunc {
	t, err := url.Parse(target)
	if err != nil {
		return func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid target"})
		}
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(t)
			r.Out.URL.Path = outboundPath
			r.Out.URL.RawPath = ""
		},
	}
	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
