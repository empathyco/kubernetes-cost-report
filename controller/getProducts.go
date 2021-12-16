package controller

import (
	"github.com/gin-gonic/gin"
	//"log"
	api "platform-cost-report/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    InstanceType	= "instance_type"
	InstanceOption 	= "instance_option"
	CPU 			= "vcpu"
	Memory 			= "memory"
	Unit 			= "unit"
	Description 	= "description"
)

func (c *Controller) GetProducts (ctx *gin.Context){


	// Gauge Vec registration 
	// https://github.com/prometheus/client_golang/issues/716#issuecomment-590282553
	// https://github.com/deathowl/go-metrics-prometheus/issues/14#issuecomment-570029311
	
    lastRequestReceivedTime := promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "instance_cost",
		Help: "Cost Instance Type",
    }, []string{InstanceType,Description, InstanceOption, CPU, Memory, Unit })


	var OnDemandPricing []api.Price

	OnDemandPricing = api.PriceMetric()

	// Exposing custom metrics 

	for i:=0; i<len(OnDemandPricing); i++ {
		lastRequestReceivedTime.With(prometheus.Labels{
			InstanceType: OnDemandPricing[i].InstanceType,
			Description: OnDemandPricing[i].Description,
			InstanceOption: "on_demand",
			CPU: OnDemandPricing[i].CPU,
			Memory: OnDemandPricing[i].Memory,
			Unit: OnDemandPricing[i].Unit,

		}).Set(OnDemandPricing[i].Price)
	}

}
