package cloud

// UnitPrice is the interface that all unit prices must implement.
type UnitPrice interface {
	GetName() string
}

// BaseUnitPrice is the base unit price for the instance type.
type BaseUnitPrice struct {
	InstanceType string
	AZ           string
	MemPrice     float64
	CPUPrice     float64
}

// OnDemand2UnitPrice is the struct for on-demand pricing.
type OnDemand2UnitPrice struct {
	BaseUnitPrice
}

// Spot2UnitPrice is the struct for the spot price.
type Spot2UnitPrice struct {
	BaseUnitPrice
	Capacity float64
}

// GetName returns the name of the instance type.
func (o OnDemand2UnitPrice) GetName() string {
	return o.InstanceType
}

// GetName returns the name of the instance type.
func (s Spot2UnitPrice) GetName() string {
	return s.InstanceType
}

// ForEach iterates over the slice and executes the function f against each element.
func ForEach(vs []UnitPrice, f func(UnitPrice)) {
	for _, v := range vs {
		f(v)
	}
}

// Index returns the index of the first instance of v in vs. If v is not present in vs, -1 is returned.
func Index(vs []UnitPrice, t UnitPrice) int {
	for i, v := range vs {
		if v.GetName() == t.GetName() {
			return i
		}
	}

	return -1
}

// Include returns true if a is in b.
func Include(vs []UnitPrice, t UnitPrice) bool {
	return Index(vs, t) >= 0
}

// Any returns true if any of the values in the slice pass the predicate p.
func Any(vs []UnitPrice, f func(UnitPrice) bool) bool {
	for _, v := range vs {
		if f(v) {
			return true
		}
	}

	return false
}

// All returns true if all of the values in the slice pass the predicate p.
func All(vs []UnitPrice, f func(UnitPrice) bool) bool {
	for _, v := range vs {
		if !f(v) {
			return false
		}
	}

	return true
}

// Filter returns a new slice containing all values where the predicate p returns true.
func Filter(vs []UnitPrice, f func(UnitPrice) bool) []UnitPrice {
	vsf := make([]UnitPrice, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}

	return vsf
}

// Intersection returns a new slice containing all values that are present in both slices.
func Intersection(vs1, vs2 []UnitPrice) []UnitPrice {
	vsint := make([]UnitPrice, 0)
	for _, v := range vs1 {
		if Include(vs2, v) {
			vsint = append(vsint, v)
		}
	}

	return vsint
}

type mapFunc func(v1, v2 UnitPrice)

// IntersectionForEach iterates over the intersection of two slices and executes the function f against each element.
func IntersectionForEach(vs1, vs2 []UnitPrice, t mapFunc) {
	for _, v := range vs1 {
		if i := Index(vs2, v); i >= 0 {
			t(v, vs2[i])
		}
	}
}
