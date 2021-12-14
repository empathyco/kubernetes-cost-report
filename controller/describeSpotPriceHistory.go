package controller

import (
	"fmt"
	"time"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Struct so that we can aggregate list of prices per instance type and AZ
type Key struct {
	InstanceType string
    AvailabilityZone string
}

func (c *Controller) DescribeSpotPriceHistory (ctx *gin.Context){
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
		return
	}

	// Var for sum of prices per instance type and AZ
	aggregatedPrices := map[Key]float64{}
	// Var to count the number of price variations per instance type and AZ
	totalPrices := map[Key]int{}
	for _, value := range result.SpotPriceHistory {
		if s, err := strconv.ParseFloat(*value.SpotPrice, 8); err == nil {
			aggregatedPrices[Key{InstanceType: *value.InstanceType, AvailabilityZone: *value.AvailabilityZone}] += s
			totalPrices[Key{InstanceType: *value.InstanceType, AvailabilityZone: *value.AvailabilityZone}] += 1
		}
    }

    // Var with "average" price per instance type and AZ
    averages := map[Key]float64{}
	for key, value := range aggregatedPrices {
		averages[key] = value / float64(totalPrices[key])
    }

    // Just print to check results
	for key, value := range averages {
		fmt.Println("-", key, value)
    }

}
