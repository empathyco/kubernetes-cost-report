package main

import (
	"flag"
	"log"
	"math/rand"
	"runtime"
	"time"
	"github.com/gin-gonic/gin"

	"cost-report/controller"
	
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	
)

var (
	indexName  string
	numWorkers int
	flushBytes int
	numItems   int
)

func init() {
	flag.StringVar(&indexName, "index", "test", "Index name")
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of indexer workers")
	flag.IntVar(&flushBytes, "flush", 5e+6, "Flush threshold in bytes")
	flag.IntVar(&numItems, "count", 10000, "Number of documents to generate")
	flag.Parse()
	prometheus.MustRegister(cpuTemp)
	rand.Seed(time.Now().UnixNano())
}



var cpuTemp = prometheus.NewGauge(prometheus.GaugeOpts{
    Name: "cpu_temperature_celsius",
    Help: "Current temperature of the CPU.",
})

func prometheusHandler() gin.HandlerFunc {
    h := promhttp.Handler()

    return func(c *gin.Context) {
        h.ServeHTTP(c.Writer, c.Request)
    }
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

	cpuTemp.Set(65.3)

	r := gin.Default()

	c := controller.NewController()

	v1 := r.Group("/api/v1")
	{
		describe := v1.Group("/describe")
		{
			describe.GET("", c.DescribeServices)
		}
		getProducts := v1.Group("/getProducts")
		{
			getProducts.GET("", c.GetProducts)
		}
	}

	cpuTemp.Set(65.3)


	r.GET("/metrics", prometheusHandler())

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