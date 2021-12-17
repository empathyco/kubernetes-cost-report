package api

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
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

func ParsingJsonFloat(dataByte []byte, key string) float64 {
	//https://github.com/tidwall/gjson

	value := gjson.Get(string(dataByte[:]), key).Float()
	return value
}

func ParsingPrice(PriceData aws.JSONValue) (*Price, error) {
	Pricing := &Price{}
	data, err := json.Marshal(PriceData)
	if err != nil {
		return nil, err
	}

	Pricing.CPU = ParsingJsonString(data, "product.attributes.vcpu")

	Pricing.InstanceType = ParsingJsonString(data, "product.attributes.instanceType")

	Pricing.Memory = ParsingJsonString(data, "product.attributes.memory")

	Pricing.Description = ParsingJsonString(data, "terms.OnDemand.*.priceDimensions.*.description")
	//Price in USD
	Pricing.Price = ParsingJsonFloat(data, "terms.OnDemand.*.priceDimensions.*.pricePerUnit.USD")

	Pricing.Unit = ParsingJsonString(data, "terms.OnDemand.*.priceDimensions.*.unit")

	return Pricing, nil
}

// Struct so that we can aggregate list of prices per instance type and AZ

// https://docs.aws.amazon.com/sdk-for-go/api/service/pricing/

// Create a Pricing service client.

func avg(array []float64) float64 {
	result := 0.0
	for _, v := range array {
		result += v
	}
	return result / float64(len(array))
}

func groupPricing(spotPrices []*ec2.SpotPrice) []Spot {
	// Var for sum of prices per instance type and AZ
	aggregatedPrices := map[Spot][]float64{}
	// Var to count the number of price variations per instance type and AZ
	for _, value := range spotPrices {
		if s, err := strconv.ParseFloat(*value.SpotPrice, 64); err == nil {
			index := Spot{InstanceType: *value.InstanceType, AZ: *value.AvailabilityZone}
			aggregatedPrices[index] = append(aggregatedPrices[index], s)
		}
	}

	// Var with "average" price per instance type and AZ
	averages := map[Spot]float64{}
	for key, value := range aggregatedPrices {
		averages[key] = avg(value)
	}

	pricesArray := []Spot{}
	// Just print to check results
	for key, value := range averages {
		var SpotOne Spot
		SpotOne.InstanceType = key.InstanceType
		SpotOne.AZ = key.AZ
		SpotOne.Price = value

		pricesArray = append(pricesArray, SpotOne)
	}
	return pricesArray
}

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
	var spotPrices []Spot
	paginator := func(page *ec2.DescribeSpotPriceHistoryOutput, b bool) bool {
		spotPrices = groupPricing(page.SpotPriceHistory)
		return b
	}
	err = svc.DescribeSpotPriceHistoryPages(input, paginator)
	if err != nil {
		return nil, err
	}

	return spotPrices, nil
}

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

	// Example iterating over at most 3 pages of a GetProducts operation.
	pageNum := 0
	var prices []*Price

	// GetProductsPages https://docs.aws.amazon.com/sdk-for-go/api/service/pricing/#Pricing.GetProductsPages
	paginator := func(page *pricing.GetProductsOutput, lastPage bool) bool {
		for _, v := range page.PriceList {
			price, err := ParsingPrice(v)
			if err != nil {
				return false
			}
			prices = append(prices, price)
		}
		// TODO: try to use lastPage
		pageNum++
		return pageNum <= 3
	}
	err = svc.GetProductsPages(input, paginator)
	if err != nil {
		return nil, err
	}
	return prices, nil
}
