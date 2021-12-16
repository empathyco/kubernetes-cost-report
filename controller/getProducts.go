package controller

import (
	"github.com/gin-gonic/gin"
	"time"
	api "platform-cost-report/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
    InstanceType	= "label_beta_kubernetes_io_instance_type"
	InstanceOption 	= "label_eks_amazonaws_com_capacity_type"
	CPU 			= "vcpu"
	Memory 			= "memory"
	Unit 			= "unit"
	Description 	= "description"
	AZ 				= "label_topology_kubernetes_io_zone"
	Region 			= "region"
	Timestamp 		= "timestamp"
)

func ExposeMetrics() prometheus.Gatherer{
	// Gauge Vec registration 
	// https://github.com/prometheus/client_golang/issues/716#issuecomment-590282553
	// https://github.com/deathowl/go-metrics-prometheus/issues/14#issuecomment-570029311
	
	reg := prometheus.NewRegistry()
    lastRequestReceivedTime := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
        Name: "instance_cost",
		Help: "Cost Instance Type",
    }, []string{InstanceType,Description, InstanceOption, CPU, Memory, Unit, AZ, Region, Timestamp })


	var OnDemandPricing []api.Price

	OnDemandPricing = api.PriceMetric()

	// Exposing custom metrics OnDemand

	for i:=0; i<len(OnDemandPricing); i++ {
		lastRequestReceivedTime.With(prometheus.Labels{
			InstanceType: OnDemandPricing[i].InstanceType,
			Description: OnDemandPricing[i].Description,
			InstanceOption: "ON_DEMAND",
			CPU: OnDemandPricing[i].CPU,
			Memory: OnDemandPricing[i].Memory,
			Unit: OnDemandPricing[i].Unit,
			AZ: "NA",
			Region: "eu-west-1",
			Timestamp: time.Now().String(),

		}).Set(OnDemandPricing[i].Price)
	}

	var SpotPricing []api.Spot 
	SpotPricing = api.SpotMetric()

	// Exposing custom metrics Spot

	for i:=0; i<len(SpotPricing); i++ {

		for j:=0; j<len(OnDemandPricing); j++ {

			if SpotPricing[i].InstanceType == OnDemandPricing[j].InstanceType {
				lastRequestReceivedTime.With(prometheus.Labels{
					InstanceType: SpotPricing[i].InstanceType,
					Description: "-",
					InstanceOption: "SPOT",
					CPU: OnDemandPricing[j].CPU,
					Memory: OnDemandPricing[j].Memory,
					Unit: "Hrs",
					AZ: SpotPricing[i].AZ,
					Region: "eu-west-1",
					Timestamp: time.Now().String(),
				}).Set(SpotPricing[i].Price)
			}
		}
		
		
	}


	return reg

}

func (c *Controller) GetProducts (ctx *gin.Context){
	ExposeMetrics()
}
