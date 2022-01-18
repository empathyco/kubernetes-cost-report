package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"platform-cost-report/cloud"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
)

func init() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
}

func main() {
	log.Printf("OS: %s\nArchitecture: %s\n", runtime.GOOS, runtime.GOARCH)

	scheduler := cron.New()

	// First exposed metrics on init
	reg, err := cloud.AWSMetrics()
	if err != nil {
		panic(err)
	}
	_, err = scheduler.AddFunc("@every 12h", func() {
		reg, err = cloud.AWSMetrics()
		fmt.Println("AWS metrics updated")
		if err != nil {
			fmt.Println("Error: %w", err)
		}
	})
	if err != nil {
		panic(err)
	}
	scheduler.Start()

	http.HandleFunc("/updatePricing", func(rw http.ResponseWriter, r *http.Request) {
		reg, err = cloud.AWSMetrics()
		if err != nil {
			fmt.Println("Error: %w", err)
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(rw, "{\"error\":\"%v\"}", err)
		}
		rw.WriteHeader(http.StatusOK)
		fmt.Fprintf(rw, "{\"message\":\"Pricing updated\"}")
	})

	http.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		fmt.Fprintf(rw, "{\"message\":\"OK\"}")
	})

	http.HandleFunc("/metrics", func(rw http.ResponseWriter, r *http.Request) {
		handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
		handler.ServeHTTP(rw, r)
	})

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}

}
