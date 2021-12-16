
package api

type Price struct {
	InstanceType string 
	Description string
	CPU string
	Memory string 
	Price float64 
	Unit string 
}

type Prices []Price
