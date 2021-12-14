package controller

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func parseTime(layout, value string) *time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}
	return &t
}

func (c *Controller) DescribeSpotPriceHistory (ctx *gin.Context){
	svc := ec2.New(session.New(), aws.NewConfig().WithRegion("us-east-1"))
	input := &ec2.DescribeSpotPriceHistoryInput{
		EndTime: parseTime("2006-01-02T15:04:05.999999999Z", "2021-12-13T00:00:00Z"),
		InstanceTypes: []*string{
			aws.String("m1.xlarge"),
		},
		ProductDescriptions: []*string{
			aws.String("Linux/UNIX (Amazon VPC)"),
		},
		StartTime: parseTime("2006-01-02T15:04:05.999999999Z", "2021-12-12T00:00:00Z"),
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

	fmt.Println(result)
}
