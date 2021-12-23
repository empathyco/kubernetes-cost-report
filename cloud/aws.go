package cloud

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"
)

type Price struct {
	InstanceType string
	Description  string
	CPU          string
	Memory       string
	Price        float64
	Unit         string
	AZ           string
	Region       string
}

type Spot struct {
	InstanceType string
	AZ           string
	Price        float64
}

// SpotUnitPrice represents the price per unit(1cpu, 1GB) of the instance type
type OnDemandUnitPrice struct {
	InstanceType string
	AZ           string
	MemPrice     float64
	CPUPrice     float64
}

type SpotUnitPrice struct {
	InstanceType string
	AZ           string
	MemPrice     float64
	CPUPrice     float64
	Capacity     float64
}

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

var filtering []*pricing.Filter = []*pricing.Filter{
	{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("PurchaseOption"),
		Value: aws.String("No Upfront"),
	},
	{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("regionCode"),
		Value: aws.String("eu-west-1"),
	},
	{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("tenancy"),
		Value: aws.String("Shared"),
	},
	{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("preInstalledSw"),
		Value: aws.String("NA"),
	},
	{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("operatingSystem"),
		Value: aws.String("Linux"),
	},
	{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("marketoption"),
		Value: aws.String("OnDemand"),
	},
}

func ParsingJsonString(dataByte []byte, key string) string {
	//https://github.com/tidwall/gjson

	value := gjson.Get(string(dataByte[:]), key).String()
	return value
}

func parsingJsonFloat(dataByte []byte, key string) float64 {
	//https://github.com/tidwall/gjson

	value := gjson.Get(string(dataByte[:]), key).Float()
	return value
}

func ParsingJsonStringArray(dataByte []byte, key string) []string {
	//https://github.com/tidwall/gjson
	result := []string{}
	value := gjson.Get(string(dataByte[:]), key).Array()
	for _, name := range value {
		result = append(result, name.String())
	}

	return result
}

func parsingPrice(PriceData aws.JSONValue) (*Price, error) {
	Pricing := &Price{}
	data, err := json.Marshal(PriceData)
	if err != nil {
		return nil, err
	}

	Pricing.CPU = ParsingJsonString(data, "product.attributes.vcpu")
	Pricing.InstanceType = ParsingJsonString(data, "product.attributes.instanceType")
	Pricing.Memory = ParsingJsonString(data, "product.attributes.memory")
	Pricing.Description = ParsingJsonString(data, "terms.OnDemand.*.priceDimensions.*.description")
	Pricing.Price = parsingJsonFloat(data, "terms.OnDemand.*.priceDimensions.*.pricePerUnit.USD")
	Pricing.Unit = ParsingJsonString(data, "terms.OnDemand.*.priceDimensions.*.unit")

	return Pricing, nil
}

// Struct so that we can aggregate list of prices per instance type and AZ

// https://docs.aws.amazon.com/sdk-for-go/api/service/pricing/

func avg(array []float64) float64 {
	result := 0.0
	for _, v := range array {
		result += v
	}
	return result / float64(len(array))
}

func (p *Price) GetCPU() int {
	cpu, _ := strconv.Atoi(p.CPU)
	return cpu
}

func (p *Price) GetMemory() int {
	num := strings.Fields(p.Memory)[0]
	memory, _ := strconv.Atoi(num)
	return memory
}
func (prices *Price) calculateOnDemandUnitPrice() OnDemandUnitPrice {
	//
	cpuMemRelation := 7.2
	gbPrice := prices.Price / (cpuMemRelation*float64(prices.GetCPU()) + float64(prices.GetMemory()))
	return OnDemandUnitPrice{
		InstanceType: prices.InstanceType,
		AZ:           prices.AZ,
		MemPrice:     gbPrice,
		CPUPrice:     cpuMemRelation * gbPrice,
	}

}

func (spot *Spot) calculateSpotUnitPrice(price *Price) SpotUnitPrice {
	//
	cpuMemRelation := 7.2
	gbPrice := price.Price / (cpuMemRelation*float64(price.GetCPU()) + float64(price.GetMemory()))
	minSpotPrice := price.Price / 5
	capacity := (spot.Price - minSpotPrice) / (4 * price.Price / 5)
	return SpotUnitPrice{
		InstanceType: price.InstanceType,
		AZ:           price.AZ,
		MemPrice:     gbPrice,
		CPUPrice:     cpuMemRelation * gbPrice,
		Capacity:     capacity,
	}

}

func groupPricing(spotPrices []*ec2.SpotPrice) []Spot {
	aggregatedPrices := map[Spot][]float64{}
	for _, value := range spotPrices {
		if s, err := strconv.ParseFloat(*value.SpotPrice, 64); err == nil {
			index := Spot{InstanceType: *value.InstanceType, AZ: *value.AvailabilityZone}
			aggregatedPrices[index] = append(aggregatedPrices[index], s)
		}
	}

	averages := map[Spot]float64{}
	for key, value := range aggregatedPrices {
		averages[key] = avg(value)
	}

	pricesArray := []Spot{}
	for key, value := range averages {
		var SpotOne Spot
		SpotOne.InstanceType = key.InstanceType
		SpotOne.AZ = key.AZ
		SpotOne.Price = value

		pricesArray = append(pricesArray, SpotOne)
	}
	return pricesArray
}

// SpotMetric is the function that returns the average spot price
func SpotMetric() ([]Spot, error) {

	ses, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := ec2.New(ses, aws.NewConfig().WithRegion("eu-west-1"))
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1)
	input := &ec2.DescribeSpotPriceHistoryInput{
		EndTime: &endTime,
		ProductDescriptions: []*string{
			aws.String("Linux/UNIX (Amazon VPC)"),
		},
		StartTime: &startTime,
	}
	var spotPrices []*ec2.SpotPrice
	paginator := func(page *ec2.DescribeSpotPriceHistoryOutput, b bool) bool {
		spotPrices = append(spotPrices, page.SpotPriceHistory...)
		return !b
	}
	err = svc.DescribeSpotPriceHistoryPages(input, paginator)
	groupPrice := groupPricing(spotPrices)
	if err != nil {
		return nil, err
	}
	return groupPrice, nil
}

// PriceMetric is the function that returns the average price
func PriceMetric() ([]*Price, error) {
	ses, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := pricing.New(ses, aws.NewConfig().WithRegion("us-east-1"))

	// GetProductsInput https://docs.aws.amazon.com/sdk-for-go/api/service/pricing/#GetProductsInput

	input := &pricing.GetProductsInput{
		Filters:     filtering,
		MaxResults:  aws.Int64(100),
		ServiceCode: aws.String("AmazonEC2"),
	}

	var prices []*Price
	// GetProductsPages https://docs.aws.amazon.com/sdk-for-go/api/service/pricing/#Pricing.GetProductsPages
	paginator := func(page *pricing.GetProductsOutput, lastPage bool) bool {
		for _, v := range page.PriceList {
			price, err := parsingPrice(v)
			if err != nil {
				return false
			}
			prices = append(prices, price)
		}
		return !lastPage
	}
	err = svc.GetProductsPages(input, paginator)
	if err != nil {
		return nil, err
	}

	return prices, nil
}

func listInstances() ([]string, error) {
	ses, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	svc := ec2.New(ses, aws.NewConfig().WithRegion("eu-west-1"))

	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("pending"),
					aws.String("running"),
					aws.String("shutting-down"),
					aws.String("terminated"),
					aws.String("stopping"),
					aws.String("stopped"),
				},
			},
		},
	}

	// Example iterating over at most 3 pages of a DescribeInstances operation.
	var instanceTypes []string
	err = svc.DescribeInstancesPages(input,
		func(page *ec2.DescribeInstancesOutput, lastPage bool) bool {
			data, _ := json.Marshal(page)
			instanceTypes = ParsingJsonStringArray(data, "Reservations.#.Instances.0.InstanceType")
			return !lastPage
		})
	if err != nil {
		return nil, err
	}
	fmt.Println(instanceTypes)
	return removeDuplicateStr(instanceTypes), nil
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func AWSMetrics() (prometheus.Gatherer, error) {

	reg := prometheus.NewRegistry()
	labelNames := []string{InstanceType, Description, InstanceOption, CPU, Memory, Unit, AZ, Region, Timestamp}
	labelUnit := []string{InstanceType, InstanceOption, Unit, AZ, Region, Timestamp}

	allMachinePricing := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_cost_all",
		Help: "Cost Instance Type",
	}, labelNames)
	inUseMachinePricing := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_cost",
		Help: "Cost Instance Type used in the account",
	}, labelNames)
	vCPUPricing := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_cpu_price",
		Help: "Cost Per vcpu and memory",
	}, labelUnit)
	memPricing := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_mem_price",
		Help: "Cost Per vcpu and memory",
	}, labelUnit)
	capacity := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_capacity",
		Help: "Capacity of the instance type",
	}, labelUnit)
	// family_price{name=c3, memory=0.02, cpu=0.01, type=ONDEMAND, zone=NA, region=eu-west-1, unit=hour}
	// family_price{name=c3, memory=0.02, cpu=0.01, type=SPOT, zone=eu-west-1a, region=eu-west-1, unit=hour}
	onDemandPricing, err := PriceMetric()
	if err != nil {
		return nil, err
	}
	spotPricing, err := SpotMetric()
	if err != nil {
		return nil, err
	}
	instanceTypes, err := listInstances()
	if err != nil {
		return nil, err
	}
	fmt.Println(instanceTypes)
	azs := [3]string{"eu-west-1a", "eu-west-1b", "eu-west-1c"}
	for _, v := range onDemandPricing {

		for _, az := range azs {
			// All machine pricing calculation
			allMachinePricing.With(prometheus.Labels{
				InstanceType:   v.InstanceType,
				Description:    v.Description,
				InstanceOption: "ON_DEMAND",
				CPU:            v.CPU,
				Memory:         v.Memory,
				Unit:           v.Unit,
				AZ:             az,
				Region:         "eu-west-1",
				Timestamp:      time.Now().String(),
			}).Set(v.Price)
			onDemandUnitPrice := v.calculateOnDemandUnitPrice()
			vCPUPricing.With(prometheus.Labels{
				InstanceType:   v.InstanceType,
				InstanceOption: "ON_DEMAND",
				Unit:           v.Unit,
				AZ:             az,
				Region:         "eu-west-1",
				Timestamp:      time.Now().String(),
			}).Set(onDemandUnitPrice.CPUPrice)
			memPricing.With(prometheus.Labels{
				InstanceType:   v.InstanceType,
				InstanceOption: "ON_DEMAND",
				Unit:           v.Unit,
				AZ:             az,
				Region:         "eu-west-1",
				Timestamp:      time.Now().String(),
			}).Set(onDemandUnitPrice.MemPrice)
		}
		// In Use machine price calculation
		for _, w := range instanceTypes {
			if w == v.InstanceType {
				for _, az := range azs {
					inUseMachinePricing.With(prometheus.Labels{
						InstanceType:   v.InstanceType,
						Description:    v.Description,
						InstanceOption: "ON_DEMAND",
						CPU:            v.CPU,
						Memory:         v.Memory,
						Unit:           v.Unit,
						AZ:             az,
						Region:         "eu-west-1",
						Timestamp:      time.Now().String(),
					}).Set(v.Price)
				}
			}

		}
	}
	// Spot machine pricing calculation
	for _, valueSpot := range spotPricing {
		// All machine pricing calculation
		for _, valueOnDemand := range onDemandPricing {
			if valueSpot.InstanceType == valueOnDemand.InstanceType {
				allMachinePricing.With(prometheus.Labels{
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
				spotUnitPrice := valueSpot.calculateSpotUnitPrice(valueOnDemand)
				vCPUPricing.With(prometheus.Labels{
					InstanceType:   valueSpot.InstanceType,
					InstanceOption: "SPOT",
					Unit:           "Hrs",
					AZ:             valueSpot.AZ,
					Region:         "eu-west-1",
					Timestamp:      time.Now().String(),
				}).Set(spotUnitPrice.CPUPrice)
				memPricing.With(prometheus.Labels{
					InstanceType:   valueSpot.InstanceType,
					InstanceOption: "SPOT",
					Unit:           "Hrs",
					AZ:             valueSpot.AZ,
					Region:         "eu-west-1",
					Timestamp:      time.Now().String(),
				}).Set(spotUnitPrice.MemPrice)
				capacity.With(prometheus.Labels{
					InstanceType:   valueSpot.InstanceType,
					InstanceOption: "SPOT",
					Unit:           "Hrs",
					AZ:             valueSpot.AZ,
					Region:         "eu-west-1",
					Timestamp:      time.Now().String(),
				}).Set(spotUnitPrice.Capacity)
				// In Use machine price calculation
				for _, w := range instanceTypes {
					if w == valueSpot.InstanceType {
						inUseMachinePricing.With(prometheus.Labels{
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
		}
	}
	return reg, nil
}
