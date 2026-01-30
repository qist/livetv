package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/qist/livetv/global"
	"github.com/qist/livetv/route"
	"github.com/qist/livetv/service"
	"github.com/robfig/cron/v3"
)

const (
	Version = "1.0.0"
)

func main() {
	// Use latest rand syntax (Go 1.20+)
	// rand.Seed is deprecated, modern Go seeds automatically

	// Set default datadir if not set
	datadir := os.Getenv("LIVETV_DATADIR")
	if datadir == "" {
		datadir = "./data"
		os.Setenv("LIVETV_DATADIR", datadir)
	}

	// Create datadir if it doesn't exist
	if err := os.MkdirAll(datadir, 0755); err != nil {
		log.Panicln("Failed to create datadir:", err)
	}

	// Configure logging based on environment variable
	var logOutput io.Writer = os.Stderr
	var logFile *os.File
	var err error

	// Check if file logging is enabled via environment variable
	if os.Getenv("LIVETV_LOG_FILE") == "1" || os.Getenv("LIVETV_LOG_FILE") == "true" {
		// Create log file with directory creation
		logFile, err = os.OpenFile(datadir+"/livetv.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Panicln("Failed to open log file:", err)
		}
		// Only output to file when file logging is enabled
		logOutput = logFile
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(logOutput)

	// Now log startup information
	log.Println("LiveTV Version", Version)
	log.Println("Server listen", os.Getenv("LIVETV_LISTEN"))
	log.Println("Server datadir", os.Getenv("LIVETV_DATADIR"))

	err = global.InitDB(datadir + "/livetv.db")
	if err != nil {
		log.Panicf("init: %s\n", err)
	}
	log.Println("LiveTV starting...")
	go service.LoadChannelCache()
	c := cron.New()
	_, err = c.AddFunc("0 */4 * * *", service.UpdateURLCache)
	if err != nil {
		log.Panicf("preloadCron: %s\n", err)
	}
	c.Start()
	sessionSecert, err := service.GetConfig("password")
	if err != nil {
		sessionSecert = "sessionSecert"
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	store := cookie.NewStore([]byte(sessionSecert))
	router.Use(sessions.Sessions("mysession", store))
	router.Static("/assert", "./assert")
	route.Register(router)
	srv := &http.Server{
		Addr:    os.Getenv("LIVETV_LISTEN"),
		Handler: router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicf("listen: %s\n", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shuting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Panicf("Server forced to shutdown: %s\n", err)
	}
	log.Println("Server exiting")
	if logFile != nil {
		logFile.Close()
	}
}
