package main

import (
	"fmt"
	"log"
	"net/http"
	"platform-cost-report/cloud"
	"runtime"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
)

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

	http.HandleFunc("/updatePricing", func(writter http.ResponseWriter, reader *http.Request) {
		reg, err = cloud.AWSMetrics()
		if err != nil {
			fmt.Println("Error: %w", err)
			writter.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(writter, "{\"error\":\"%v\"}", err)

			return
		}
		writter.WriteHeader(http.StatusOK)
		fmt.Fprintf(writter, "{\"message\":\"Pricing updated\"}")
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
