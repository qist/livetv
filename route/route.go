package route

import (
	"github.com/gin-gonic/gin"
	"github.com/qist/livetv/handler"
	"github.com/qist/livetv/service"
)

func Register(r *gin.Engine) {
	r.LoadHTMLGlob("view/*")

	// Register default M3U route
	r.GET("/lives.m3u", handler.M3UHandler)

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

		// Check if this filename matches the configured M3U filename
		m3uFilename, err := service.GetConfig("m3u_filename")
		if err == nil && filename == m3uFilename && filename != "lives.m3u" {
			// Handle M3U request
			handler.M3UHandler(c)
			c.Abort()
			return
		}

		c.Next()
	})

	r.GET("/live.m3u8", handler.LiveHandler)
	r.GET("/live.ts", handler.TsProxyHandler)
	r.GET("/cache.txt", handler.CacheHandler)

	r.GET("/", handler.IndexHandler)
	r.POST("/api/newchannel", handler.NewChannelHandler)
	r.GET("/api/delchannel", handler.DeleteChannelHandler)
	r.POST("/api/updconfig", handler.UpdateConfigHandler)
	r.GET("/log", handler.LogHandler)
	r.GET("/login", handler.LoginViewHandler)
	r.POST("/api/login", handler.LoginActionHandler)
	r.GET("/api/logout", handler.LogoutHandler)
	r.POST("/api/changepwd", handler.ChangePasswordHandler)
}
