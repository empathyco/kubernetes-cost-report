
package api

type Price struct {
	InstanceType string 
	Description string
	CPU string
	Memory string 
	Price float64 
	Unit string 
	AZ string
	Region string
}

type Prices []Price

type Spot struct {
	InstanceType string
    AZ string
	Price float64
}

type PricesSpot []Spot