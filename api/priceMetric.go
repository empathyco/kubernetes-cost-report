
package api

import (
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
	
	 