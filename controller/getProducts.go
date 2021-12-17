package controller

import (
	api "platform-cost-report/api"
	"time"

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

	onDemandPricing, err := api.PriceMetric()
	if err != nil {
		return nil, err
	}

	// Exposing custom metrics OnDemandPricing
	for _, v := range onDemandPricing {
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

	spotPricing, err := api.SpotMetric()
	if err != nil {
		return nil, err
	}

	// Exposing custom metrics SpotPricing
	for _, valueSpot := range spotPricing {
		for _, valueOnDemand := range onDemandPricing {
			if valueSpot.InstanceType == valueOnDemand.InstanceType {
				lastRequestReceivedTime.With(prometheus.Labels{
					InstanceType:   valueSpot.InstanceType,
					Description:    "-",
					InstanceOption: "SPOT",
					CPU:            valueOnDemand.CPU,
					Memory:         valueOnDemand.Memory,
					Unit:           "Hrs",
					AZ:             valueSpot.AZ,
					Region:         "eu-west-1",
					Timestamp:      time.Now().String(),
				}).Set(valueSpot.Price)
			}
		}
	}
	return reg, nil
}
