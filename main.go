package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"platform-cost-report/cloud"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func init() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
}

// @title           Swagger Example API
// @version         1.0
// @description     This is a sample server celler server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /health

// @securityDefinitions.basic  BasicAuth

func main() {
	log.Printf("OS: %s\nArchitecture: %s\n", runtime.GOOS, runtime.GOARCH)

	scheduler := cron.New()

	r := gin.Default()

	// First exposed metrics on init
	// TODO: move to separate init method
	reg, err := cloud.AWSMetrics()
	if err != nil {
		panic(err)
	}
	scheduler.AddFunc("@every 12h", func() {
		reg, err = cloud.AWSMetrics()
		fmt.Println("AWS metrics updated")
		if err != nil {
			fmt.Println("Error: %w", err)
		}
	})
	scheduler.Start()

	r.GET("/updatePricing", func(c *gin.Context) {
		reg, err = cloud.AWSMetrics()
		if err != nil {
			fmt.Println("Error: %w", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(200, reg)
	})
	// Metrics handler
	r.GET("/metrics", func(c *gin.Context) {
		handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
		handler.ServeHTTP(c.Writer, c.Request)
	})
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "health",
		})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")

}
