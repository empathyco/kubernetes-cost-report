package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	co2 "github.com/empathyco/platform-co2-exporter/pkg"
	"k8s.io/client-go/util/homedir"
)

func newMetrics(ctx context.Context) *co2.Metrics {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	// use the current context in kubeconfig
	k8s := co2.NewMetrics(ctx, kubeconfig)
	// fmt.Println(k8s.PrintNodes())
	// fmt.Println(k8s.PrintPods())

	return k8s
}

func main() {
	ctx := context.Background()
	co2DB := co2.NewCo2DB(ctx)
	metrics := newMetrics(ctx)
	http.HandleFunc("/updateco2", func(write http.ResponseWriter, read *http.Request) {
		if err := co2DB.UpdateData(); err != nil {
			write.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(write, "{\"message\":\"%s\"}", err)

			return
		}
		write.WriteHeader(http.StatusOK)
		fmt.Fprintf(write, "{\"message\":\"OK\"}")
	})

	http.HandleFunc("/getCo2Nodes", func(write http.ResponseWriter, read *http.Request) {
		if err := metrics.GetMetrics(); err != nil {
			write.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(write, "{\"message\":\"%s\"}", err)

			return
		}
		co2NodesMetrics := co2DB.GetNodesConsumption(metrics)
		data, _ := json.Marshal(&co2NodesMetrics)
		write.WriteHeader(http.StatusOK)
		write.Header().Set("Content-Type", "application/json")
		fmt.Fprint(write, string(data))
	})

	http.HandleFunc("/getCo2Pods", func(write http.ResponseWriter, read *http.Request) {
		if err := metrics.GetMetrics(); err != nil {
			write.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(write, "{\"message\":\"%s\"}", err)

			return
		}
		co2PodsMetrics := co2DB.GetPodsConsumption(metrics)
		data, _ := json.Marshal(&co2PodsMetrics)
		write.WriteHeader(http.StatusOK)
		write.Header().Set("Content-Type", "application/json")
		fmt.Fprint(write, string(data))
	})
	http.HandleFunc("/metrics", func(write http.ResponseWriter, read *http.Request) {
		if err := metrics.GetMetrics(); err != nil {
			write.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(write, "{\"message\":\"%s\"}", err)

			return
		}
		reg := co2DB.GetMetrics(metrics)
		handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
		handler.ServeHTTP(write, read)
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
