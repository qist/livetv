package route

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/qist/livetv/handler"
	"github.com/qist/livetv/service"
)

func Register(r *gin.Engine) {
	r.LoadHTMLGlob("view/*")

	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if !requiresTokenCheck(path) {
			c.Next()
			return
		}
		enabled, param, expected := loadTokenConfig()
		if !enabled {
			c.Next()
			return
		}
		if c.Query(param) != expected {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})

	// Register default M3U route
	r.GET("/lives.m3u", handler.M3UHandler)
	r.GET("/lives.txt", handler.TxtHandler)

	// Add middleware to handle configurable M3U filename
	r.Use(func(c *gin.Context) {
		// Get the request path
		path := c.Request.URL.Path
		if path == "/" {
			c.Next()
			return
		}

		// Remove leading slash to get filename
		filename := path[1:]
		if filename == "" {
			c.Next()
			return
		}

		// Check if this filename matches the configured playlist base name
		m3uFilenameValue, err := service.GetConfig("m3u_filename")
		if err == nil {
			m3uFilename := service.DeriveM3UFilename(m3uFilenameValue)
			if filename == m3uFilename && filename != "lives.m3u" {
				// Handle M3U request
				handler.M3UHandler(c)
				c.Abort()
				return
			}
			txtFilename := service.DeriveTxtFilename(m3uFilenameValue)
			if filename == txtFilename && filename != "lives.txt" {
				handler.TxtHandler(c)
				c.Abort()
				return
			}
		}

		c.Next()
	})

	r.GET("/live.m3u8", handler.LiveHandler)
	r.GET("/live.ts", handler.TsProxyHandler)
	r.GET("/cache.txt", handler.CacheHandler)

	r.GET("/", handler.IndexHandler)
	r.POST("/api/newchannel", handler.NewChannelHandler)
	r.POST("/api/updchannel", handler.UpdateChannelHandler)
	r.GET("/api/delchannel", handler.DeleteChannelHandler)
	r.POST("/api/updconfig", handler.UpdateConfigHandler)
	r.GET("/log", handler.LogHandler)
	r.GET("/login", handler.LoginViewHandler)
	r.POST("/api/login", handler.LoginActionHandler)
	r.GET("/api/logout", handler.LogoutHandler)
	r.POST("/api/changepwd", handler.ChangePasswordHandler)
}

func requiresTokenCheck(path string) bool {
	if path == "/live.m3u8" || path == "/live.ts" || path == "/cache.txt" || path == "/lives.m3u" || path == "/lives.txt" {
		return true
	}
	filename := strings.TrimPrefix(path, "/")
	if filename == "" {
		return false
	}
	m3uFilenameValue, err := service.GetConfig("m3u_filename")
	if err != nil {
		return false
	}
	m3uFilename := service.DeriveM3UFilename(m3uFilenameValue)
	if filename == m3uFilename {
		return true
	}
	txtFilename := service.DeriveTxtFilename(m3uFilenameValue)
	return filename == txtFilename
}

func loadTokenConfig() (enabled bool, param string, expected string) {
	if v, err := service.GetConfig("token_enabled"); err == nil {
		v = strings.TrimSpace(strings.ToLower(v))
		enabled = v == "1" || v == "true" || v == "yes" || v == "on"
	}
	param, err := service.GetConfig("token_param")
	if err != nil {
		param = "token"
	}
	param = strings.TrimSpace(param)
	if param == "" {
		param = "token"
	}
	expected, err = service.GetConfig("token_value")
	if err != nil {
		expected = "livetv"
	}
	expected = strings.TrimSpace(expected)
	if expected == "" {
		expected = "livetv"
	}
	return enabled, param, expected
}
