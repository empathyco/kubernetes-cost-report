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
	fmt.Println(k8s.PrintPods())
	return k8s
}

func main() {
	ctx := context.Background()
	db := co2.NewCo2DB(ctx)
	metrics := newMetrics(ctx)

	// updateData(ctx, metrics)
	http.HandleFunc("/updateco2", func(rw http.ResponseWriter, r *http.Request) {
		db.UpdateData()
		rw.WriteHeader(http.StatusOK)
		fmt.Fprintf(rw, "{\"message\":\"OK\"}")
	})

	http.HandleFunc("/getCo2Nodes", func(rw http.ResponseWriter, r *http.Request) {
		if err := metrics.GetMetrics(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(rw, "{\"message\":\"%s\"}", err)
			return
		}

		co2NodesMetrics := db.GetNodesConsumption(metrics)
		data, _ := json.Marshal(&co2NodesMetrics)
		rw.WriteHeader(http.StatusOK)
		rw.Header().Set("Content-Type", "application/json")
		fmt.Fprint(rw, string(data))
	})

	http.HandleFunc("/getCo2Pods", func(rw http.ResponseWriter, r *http.Request) {
		if err := metrics.GetMetrics(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(rw, "{\"message\":\"%s\"}", err)
			return
		}

		co2PodsMetrics := db.GetPodsConsumption(metrics)
		data, _ := json.Marshal(&co2PodsMetrics)
		rw.WriteHeader(http.StatusOK)
		rw.Header().Set("Content-Type", "application/json")
		fmt.Fprint(rw, string(data))
	})
	http.HandleFunc("/metrics", func(rw http.ResponseWriter, r *http.Request) {
		if err := metrics.GetMetrics(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(rw, "{\"message\":\"%s\"}", err)
			return
		}
		reg := db.GetMetrics(metrics)
		handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
		handler.ServeHTTP(rw, r)
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
