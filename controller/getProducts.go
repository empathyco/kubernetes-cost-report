package controller

import (
	"fmt"
	api "platform-cost-report/api"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	InstanceType   = "label_beta_kubernetes_io_instance_type"
	InstanceOption = "label_eks_amazonaws_com_capacity_type"
	CPU            = "vcpu"
	Memory         = "memory"
	Unit           = "unit"
	Description    = "description"
	AZ             = "label_topology_kubernetes_io_zone"
	Region         = "region"
	Timestamp      = "timestamp"
)

func ExposeMetrics() (prometheus.Gatherer, error) {
	// Gauge Vec registration
	// https://github.com/prometheus/client_golang/issues/716#issuecomment-590282553
	// https://github.com/deathowl/go-metrics-prometheus/issues/14#issuecomment-570029311

	reg := prometheus.NewRegistry()
	labelNames := []string{InstanceType, Description, InstanceOption, CPU, Memory, Unit, AZ, Region, Timestamp}
	lastRequestReceivedTime := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_cost",
		Help: "Cost Instance Type",
	}, labelNames)

	onDemandPricing := api.PriceMetric()

	// Exposing custom metrics OnDemand

	for _, v := range OnDemandPricing {
		lastRequestReceivedTime.With(prometheus.Labels{
			InstanceType:   v.InstanceType,
			Description:    v.Description,
			InstanceOption: "ON_DEMAND",
			CPU:            v.CPU,
			Memory:         v.Memory,
			Unit:           v.Unit,
			AZ:             v.AZ,
			Region:         v.Region,
			Timestamp:      time.Now().String(),
		}).Set(v.Price)
	}

	SpotPricing, err := api.SpotMetric()
	if err != nil {
		return nil, err
	}

	// Exposing custom metrics Spot

	for i := 0; i < len(SpotPricing); i++ {

		for j := 0; j < len(OnDemandPricing); j++ {

			if SpotPricing[i].InstanceType == OnDemandPricing[j].InstanceType {
				lastRequestReceivedTime.With(prometheus.Labels{
					InstanceType:   SpotPricing[i].InstanceType,
					Description:    "-",
					InstanceOption: "SPOT",
					CPU:            OnDemandPricing[j].CPU,
					Memory:         OnDemandPricing[j].Memory,
					Unit:           "Hrs",
					AZ:             SpotPricing[i].AZ,
					Region:         "eu-west-1",
					Timestamp:      time.Now().String(),
				}).Set(SpotPricing[i].Price)
			}
		}

	}

	fmt.Println("Exposing metrics")

	return reg, nil

}

func (c *Controller) GetProducts(ctx *gin.Context) {

	ExposeMetrics()

}
