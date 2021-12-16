
package api

import (
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"time"
	"strconv"
	"fmt"
	"encoding/json"
	"github.com/tidwall/gjson"
)

var filtering []*pricing.Filter = []*pricing.Filter{
	{
		Type: aws.String("TERM_MATCH"),
		Field: aws.String("PurchaseOption"),
		Value: aws.String("No Upfront"),
	},
	{
		Type: aws.String("TERM_MATCH"),
		Field: aws.String("regionCode"),
		Value: aws.String("eu-west-1"),
	},
	{
		Type: aws.String("TERM_MATCH"),
		Field: aws.String("tenancy"),
		Value: aws.String("Shared"),
	},
	{
		Type: aws.String("TERM_MATCH"),
		Field: aws.String("preInstalledSw"),
		Value: aws.String("NA"),
	},
	{
		Type: aws.String("TERM_MATCH"),
		Field: aws.String("operatingSystem"),
		Value: aws.String("Linux"),
	},
	{
		Type: aws.String("TERM_MATCH"),
		Field: aws.String("marketoption"),
		Value: aws.String("OnDemand"),
	},	
}

func ParsingJsonString (dataByte []byte, key string) string{
	//https://github.com/tidwall/gjson

	value := gjson.Get(string(dataByte[:]), key).String() 
	return value 
}

func ParsingJsonFloat (dataByte []byte, key string) float64{
	//https://github.com/tidwall/gjson

	value := gjson.Get(string(dataByte[:]), key).Float() 
	return value 
}

func ParsingPrice(PriceData aws.JSONValue) Price {
	var Pricing Price
	data, err := json.Marshal(PriceData)
	if err != nil {
		fmt.Printf("marshal failed: %s", err)
	}

	Pricing.CPU = ParsingJsonString(data, "product.attributes.vcpu") 
		
	Pricing.InstanceType = ParsingJsonString(data, "product.attributes.instanceType")
	
	Pricing.Memory = ParsingJsonString(data,"product.attributes.memory")

	Pricing.Description = ParsingJsonString(data, "terms.OnDemand.*.priceDimensions.*.description")
	//Price in USD
	Pricing.Price = ParsingJsonFloat(data, "terms.OnDemand.*.priceDimensions.*.pricePerUnit.USD")

	Pricing.Unit = ParsingJsonString(data, "terms.OnDemand.*.priceDimensions.*.unit" )

	return Pricing
}

// Struct so that we can aggregate list of prices per instance type and AZ


func SpotMetric() []Spot {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeSpotPriceHistory
	svc := ec2.New(session.New(), aws.NewConfig().WithRegion("eu-west-1"))
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1)
	input := &ec2.DescribeSpotPriceHistoryInput{
		EndTime: &endTime,
		ProductDescriptions: []*string{
			aws.String("Linux/UNIX (Amazon VPC)"),
		},
		StartTime: &startTime,
	}

	result, err := svc.DescribeSpotPriceHistory(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil
	}

	//fmt.Println(result)

	// Var for sum of prices per instance type and AZ
	aggregatedPrices := map[Spot]float64{}
	// Var to count the number of price variations per instance type and AZ
	totalPrices := map[Spot]int{}
	for _, value := range result.SpotPriceHistory {
		if s, err := strconv.ParseFloat(*value.SpotPrice, 8); err == nil {
			aggregatedPrices[Spot{InstanceType: *value.InstanceType, AZ: *value.AvailabilityZone}] += s
			totalPrices[Spot{InstanceType: *value.InstanceType, AZ: *value.AvailabilityZone}] += 1
		}
    }

    // Var with "average" price per instance type and AZ
    averages := map[Spot]float64{}

	
	for key, value := range aggregatedPrices {
		averages[key] = value / float64(totalPrices[key])
    }
	
	var PricesArray PricesSpot
    // Just print to check results
	for key, value := range averages {
		var SpotOne Spot
		//fmt.Println("-", key, value)
		SpotOne.InstanceType = key.InstanceType
		SpotOne.AZ = key.AZ 
		SpotOne.Price = value

		PricesArray = append(PricesArray, SpotOne)
    }
	//fmt.Println(PricesArray)
	return PricesArray
}

func PriceMetric() []Price{
	svc := pricing.New(session.New(), aws.NewConfig().WithRegion("us-east-1"))

	// GetProductsInput https://docs.aws.amazon.com/sdk-for-go/api/service/pricing/#GetProductsInput

	input := &pricing.GetProductsInput{
		Filters:  		filtering,
		MaxResults:    	aws.Int64(100),
		ServiceCode:   	aws.String("AmazonEC2"),
	}

	// Example iterating over at most 3 pages of a GetProducts operation.
	pageNum := 0
	var PriceArray Prices

	// GetProductsPages https://docs.aws.amazon.com/sdk-for-go/api/service/pricing/#Pricing.GetProductsPages

	err := svc.GetProductsPages(input,
		func(page *pricing.GetProductsOutput, lastPage bool) bool {
			pageNum++
			for i:= 0; i<len(page.PriceList); i++ {
				
				PriceOne := ParsingPrice(page.PriceList[i])

				PriceArray = append(PriceArray, PriceOne)
			
			} 
			return pageNum <= 3
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case pricing.ErrCodeInternalErrorException:
					fmt.Println(pricing.ErrCodeInternalErrorException, aerr.Error())
				case pricing.ErrCodeInvalidParameterException:
					fmt.Println(pricing.ErrCodeInvalidParameterException, aerr.Error())
				case pricing.ErrCodeNotFoundException:
					fmt.Println(pricing.ErrCodeNotFoundException, aerr.Error())
				case pricing.ErrCodeInvalidNextTokenException:
					fmt.Println(pricing.ErrCodeInvalidNextTokenException, aerr.Error())
				case pricing.ErrCodeExpiredNextTokenException:
					fmt.Println(pricing.ErrCodeExpiredNextTokenException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
			return nil
		}

		return PriceArray
}
	
	 