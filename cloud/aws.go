package cloud

import (
	"encoding/json"
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

// OnDemandUnitPrice represents the price per unit(1cpu, 1GB) of the instance type
type OnDemandUnitPrice struct {
	InstanceType string
	AZ           string
	MemPrice     float64
	CPUPrice     float64
}

type SpotUnitPrice struct {
	OnDemandUnitPrice
	Capacity float64
	Discount float64
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
	cpuMemRelation = 7.2
)

var (
	filtering []*pricing.Filter = []*pricing.Filter{
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
)

func ParsingJSONString(dataByte []byte, key string) string {
	value := gjson.Get(string(dataByte[:]), key).String()
	return value
}

func parsingJSONFloat(dataByte []byte, key string) float64 {
	value := gjson.Get(string(dataByte[:]), key).Float()
	return value
}

func parsingJSONStringArray(dataByte []byte, key string) []string {
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

	Pricing.CPU = ParsingJSONString(data, "product.attributes.vcpu")
	Pricing.InstanceType = ParsingJSONString(data, "product.attributes.instanceType")
	Pricing.Memory = ParsingJSONString(data, "product.attributes.memory")
	Pricing.Price = parsingJSONFloat(data, "terms.OnDemand.*.priceDimensions.*.pricePerUnit.USD")
	Pricing.Unit = ParsingJSONString(data, "terms.OnDemand.*.priceDimensions.*.unit")

	return Pricing, nil
}

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

func (p *Price) CalcUnitPrice() OnDemandUnitPrice {
	gbPrice := p.Price / (cpuMemRelation*float64(p.GetCPU()) + float64(p.GetMemory()))
	return OnDemandUnitPrice{
		InstanceType: p.InstanceType,
		AZ:           p.AZ,
		MemPrice:     gbPrice,
		CPUPrice:     cpuMemRelation * gbPrice,
	}
}

func (spot *Spot) CalcUnitPrice(valuespot Spot, price *Price) SpotUnitPrice {
	// Considering the cpuMemRelation is a constant
	gbPrice := valuespot.Price / (cpuMemRelation*float64(price.GetCPU()) + float64(price.GetMemory()))
	// Min Spo Price is around a 80% of saving for the OnDemand price
	minSpotPrice := price.Price / 5
	discount := 1 - valuespot.Price/price.Price
	//fmt.Println(discount)
	// Spot capacity for the instanceType based on the pricing
	capacity := (spot.Price - minSpotPrice) / (4 * price.Price / 5)
	return SpotUnitPrice{
		OnDemandUnitPrice: OnDemandUnitPrice{
			InstanceType: valuespot.InstanceType,
			AZ:           valuespot.AZ,
			MemPrice:     gbPrice,
			CPUPrice:     cpuMemRelation * gbPrice,
		},
		Capacity: capacity,
		Discount: discount,
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
		spotOne := Spot{
			InstanceType: key.InstanceType,
			AZ:           key.AZ,
			Price:        value,
		}
		pricesArray = append(pricesArray, spotOne)
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
			instanceTypes = parsingJSONStringArray(data, "Reservations.#.Instances.0.InstanceType")
			return !lastPage
		})
	if err != nil {
		return nil, err
	}
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
	labelNames := []string{InstanceType, InstanceOption, CPU, Memory, Unit, AZ, Region}
	labelUnit := []string{InstanceType, InstanceOption, Unit, AZ, Region}

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
	discount := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_discount",
		Help: "Discount of the instance type",
	}, labelUnit)

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

	for _, v := range onDemandPricing {
		onDemandUnitPrice := v.CalcUnitPrice()
		// All machine pricing calculation
		allMachinePricing.With(prometheus.Labels{
			InstanceType:   v.InstanceType,
			InstanceOption: "ON_DEMAND",
			CPU:            v.CPU,
			Memory:         v.Memory,
			Unit:           v.Unit,
			AZ:             "",
			Region:         "eu-west-1",
		}).Set(v.Price)
		vCPUPricing.With(prometheus.Labels{
			InstanceType:   v.InstanceType,
			InstanceOption: "ON_DEMAND",
			Unit:           v.Unit,
			AZ:             "",
			Region:         "eu-west-1",
		}).Set(onDemandUnitPrice.CPUPrice)
		memPricing.With(prometheus.Labels{
			InstanceType:   v.InstanceType,
			InstanceOption: "ON_DEMAND",
			Unit:           v.Unit,
			AZ:             "",
			Region:         "eu-west-1",
		}).Set(onDemandUnitPrice.MemPrice)
		// In Use machine price calculation
		for _, w := range instanceTypes {
			if w == v.InstanceType {
				inUseMachinePricing.With(prometheus.Labels{
					InstanceType:   v.InstanceType,
					InstanceOption: "ON_DEMAND",
					CPU:            v.CPU,
					Memory:         v.Memory,
					Unit:           v.Unit,
					AZ:             "",
					Region:         "eu-west-1",
				}).Set(v.Price)
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
					InstanceOption: "SPOT",
					CPU:            valueOnDemand.CPU,
					Memory:         valueOnDemand.Memory,
					Unit:           "Hrs",
					AZ:             valueSpot.AZ,
					Region:         "eu-west-1",
				}).Set(valueSpot.Price)
				spotUnitPrice := valueSpot.CalcUnitPrice(valueSpot, valueOnDemand)
				vCPUPricing.With(prometheus.Labels{
					InstanceType:   valueSpot.InstanceType,
					InstanceOption: "SPOT",
					Unit:           "Hrs",
					AZ:             valueSpot.AZ,
					Region:         "eu-west-1",
				}).Set(spotUnitPrice.CPUPrice)
				memPricing.With(prometheus.Labels{
					InstanceType:   valueSpot.InstanceType,
					InstanceOption: "SPOT",
					Unit:           "Hrs",
					AZ:             valueSpot.AZ,
					Region:         "eu-west-1",
				}).Set(spotUnitPrice.MemPrice)
				capacity.With(prometheus.Labels{
					InstanceType:   valueSpot.InstanceType,
					InstanceOption: "SPOT",
					Unit:           "Hrs",
					AZ:             valueSpot.AZ,
					Region:         "eu-west-1",
				}).Set(spotUnitPrice.Capacity)
				discount.With(prometheus.Labels{
					InstanceType:   valueSpot.InstanceType,
					InstanceOption: "SPOT",
					Unit:           "Hrs",
					AZ:             valueSpot.AZ,
					Region:         "eu-west-1",
				}).Set(spotUnitPrice.Discount)
				// In Use machine price calculation
				for _, w := range instanceTypes {
					if w == valueSpot.InstanceType {
						inUseMachinePricing.With(prometheus.Labels{
							InstanceType:   valueSpot.InstanceType,
							InstanceOption: "SPOT",
							CPU:            valueOnDemand.CPU,
							Memory:         valueOnDemand.Memory,
							Unit:           "Hrs",
							AZ:             valueSpot.AZ,
							Region:         "eu-west-1",
						}).Set(valueSpot.Price)
					}
				}
			}
		}
	}
	return reg, nil
}
