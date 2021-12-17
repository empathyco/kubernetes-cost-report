package api

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

func TestParsingJsonString(t *testing.T) {
	type args struct {
		dataByte []byte
		key      string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			name: "Test Parse Json String",
			args: args{
				dataByte: []byte(`{"key":"value"}`),
				key:      "key",
			},
			want: "value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParsingJsonString(tt.args.dataByte, tt.args.key); got != tt.want {
				t.Errorf("ParsingJsonString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsingJsonFloat(t *testing.T) {
	type args struct {
		dataByte []byte
		key      string
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		// TODO: Add test cases.
		{
			name: "Test Parse Json Float",
			args: args{
				dataByte: []byte(`{"key":1.0}`),
				key:      "key",
			},
			want: 1.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParsingJsonFloat(tt.args.dataByte, tt.args.key); got != tt.want {
				t.Errorf("ParsingJsonFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsingPrice(t *testing.T) {
	type args struct {
		PriceData aws.JSONValue
	}
	tests := []struct {
		name string
		args args
		want *Price
	}{
		// TODO: Add test cases.
		{
			name: "Test Parsing Price",
			args: args{
				PriceData: aws.JSONValue{
					"product": aws.JSONValue{
						"attributes": aws.JSONValue{
							"vcpu":         "1.0",
							"instanceType": "t2.micro",
							"memory":       "1.0",
						},
					},
				},
			},
			want: &Price{
				InstanceType: "t2.micro",
				CPU:          "1.0",
				Memory:       "1.0",
				Price:        2.0,
				Unit:         "Hrs",
				Description:  "Linux/UNIX (Amazon VPC)",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParsingPrice(tt.args.PriceData); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsingPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}
