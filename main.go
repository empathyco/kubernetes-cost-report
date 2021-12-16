package main

import (
	"flag"
	"log"
	"math/rand"
	"runtime"
	"time"
	"github.com/gin-gonic/gin"

	"platform-cost-report/controller"
	
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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


	r := gin.Default()

	c := controller.NewController()

	v1 := r.Group("/api/v1")
	{
		getProducts := v1.Group("/getProducts")
		{
			getProducts.GET("", c.GetProducts)
		}
	}

    // Metrics handler
    r.GET("/metrics", func(c *gin.Context) {
        handler := promhttp.Handler()
        handler.ServeHTTP(c.Writer, c.Request)
    })

	health := r.Group("/health")
	{
		health.GET("", func(c *gin.Context) {
			
			c.JSON(200, gin.H{
				"status": "health",
			})
		})
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}