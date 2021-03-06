package cloud

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/prometheus/client_golang/prometheus"
)

func skipCI(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
}

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
			if got := parsingJSONString(tt.args.dataByte, tt.args.key); got != tt.want {
				t.Errorf("ParsingJsonString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsingJsonStringArray(t *testing.T) {
	type args struct {
		dataByte []byte
		key      string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Test Parse Json Float",
			args: args{
				dataByte: []byte(`{"key":["test","test2"]}`),
				key:      "key",
			},
			want: []string{"test", "test2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsingJSONStringArray(tt.args.dataByte, tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsingJsonStringArray() = %v, want %v", got, tt.want)
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
			if got := parsingJSONFloat(tt.args.dataByte, tt.args.key); got != tt.want {
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
					"terms": aws.JSONValue{
						"OnDemand": aws.JSONValue{
							"1.0": aws.JSONValue{
								"priceDimensions": aws.JSONValue{
									"t2.micro": aws.JSONValue{
										"pricePerUnit": aws.JSONValue{
											"USD": "2.00",
										},
										"description": "Linux/UNIX (Amazon VPC)",
										"unit":        "Hrs",
									},
								},
							},
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsingPrice(tt.args.PriceData)
			if err != nil {
				t.Errorf("ParsingPrice() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsingPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_avg(t *testing.T) {
	type args struct {
		array []float64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "Test Avg",
			args: args{
				array: []float64{1.0, 2.0, 3.0},
			},
			want: 2.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := avg(tt.args.array); got != tt.want {
				t.Errorf("avg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_groupPricing(t *testing.T) {
	type args struct {
		spotPrices []*ec2.SpotPrice
	}
	tests := []struct {
		name string
		args args
		want []Spot
	}{
		{
			name: "Test Group Pricing 1",
			args: args{
				spotPrices: []*ec2.SpotPrice{
					{
						AvailabilityZone:   aws.String("us-east-1a"),
						InstanceType:       aws.String("t2.micro"),
						ProductDescription: aws.String("Linux/UNIX (Amazon VPC)"),
						SpotPrice:          aws.String("0.01"),
						Timestamp:          aws.Time(time.Now()),
					},
					{
						AvailabilityZone:   aws.String("us-east-1a"),
						InstanceType:       aws.String("t2.micro"),
						ProductDescription: aws.String("Linux/UNIX (Amazon VPC)"),
						SpotPrice:          aws.String("0.02"),
						Timestamp:          aws.Time(time.Now()),
					},
				},
			},
			want: []Spot{
				{
					InstanceType: "t2.micro",
					AZ:           "us-east-1a",
					Price:        0.015,
				},
			},
		},
		{
			name: "Test Group Pricing 2",
			args: args{
				spotPrices: []*ec2.SpotPrice{
					{
						AvailabilityZone:   aws.String("us-east-1b"),
						InstanceType:       aws.String("t2.micro"),
						ProductDescription: aws.String("Linux/UNIX (Amazon VPC)"),
						SpotPrice:          aws.String("0.03"),
						Timestamp:          aws.Time(time.Now()),
					},
				},
			},
			want: []Spot{
				{
					InstanceType: "t2.micro",
					AZ:           "us-east-1b",
					Price:        0.03,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := groupPricing(tt.args.spotPrices); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupPricing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSpotMetric(t *testing.T) {
	skipCI(t)
	tests := []struct {
		name    string
		want    bool
		wantErr bool
	}{
		{
			name: "Test Spot Metric",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SpotMetric()
			if (err != nil) != tt.wantErr {
				t.Errorf("SpotMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) > 0 != tt.want {
				t.Errorf("SpotMetric() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriceMetric(t *testing.T) {
	skipCI(t)
	tests := []struct {
		name    string
		want    bool
		wantErr bool
	}{
		{
			name: "Test Price Metric",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PriceMetric()
			if (err != nil) != tt.wantErr {
				t.Errorf("PriceMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			test := got[0]
			if !reflect.DeepEqual(len(got) > 0, tt.want) {
				t.Errorf("PriceMetric() = %v, want %v", test, tt.want)
			}
		})
	}
}

func TestAWSMetrics(t *testing.T) {
	skipCI(t)
	tests := []struct {
		name    string
		want    prometheus.Gatherer
		wantErr bool
	}{
		{
			name: "Test AWS Metrics",
			want: prometheus.Gatherer(prometheus.NewRegistry()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AWSMetrics()
			if (err != nil) != tt.wantErr {
				t.Errorf("AWSMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}

func Test_listInstances(t *testing.T) {
	skipCI(t)
	tests := []struct {
		name    string
		want    []string
		wantErr bool
	}{
		{
			name:    "Test List Instances",
			want:    []string{"i-0f8f8f8f8f8f8f8f8", "i-0f8f8f8f8f8f8f8f8"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := listInstances()
			if (err != nil) != tt.wantErr {
				t.Errorf("listInstances() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) == 0 {
				t.Errorf("listInstances() = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func Test_removeDuplicateStr(t *testing.T) {
	type args struct {
		strSlice []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Test remove duplicate string",
			args: args{strSlice: []string{"us-east-1a", "us-east-1b", "us-east-1a"}},
			want: []string{"us-east-1a", "us-east-1b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeDuplicateStr(tt.args.strSlice); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("removeDuplicateStr() = %v, want %v", got, tt.want)
			}
		})
	}
}
