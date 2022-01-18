package cloud

type UnitPrice interface {
	// Methods
	GetName() string
}

type BaseUnitPrice struct {
	InstanceType string
	AZ           string
	MemPrice     float64
	CPUPrice     float64
}

type OnDemand2UnitPrice struct {
	BaseUnitPrice
}

type Spot2UnitPrice struct {
	BaseUnitPrice
	Capacity float64
}

func (o OnDemand2UnitPrice) GetName() string {
	return o.InstanceType
}

func (s Spot2UnitPrice) GetName() string {
	return s.InstanceType
}

func ForEach(vs []UnitPrice, f func(UnitPrice)) {
	for _, v := range vs {
		f(v)
	}
}

func Index(vs []UnitPrice, t UnitPrice) int {
	for i, v := range vs {
		if v.GetName() == t.GetName() {
			return i
		}
	}
	return -1
}

func Include(vs []UnitPrice, t UnitPrice) bool {
	return Index(vs, t) >= 0
}

func Any(vs []UnitPrice, f func(UnitPrice) bool) bool {
	for _, v := range vs {
		if f(v) {
			return true
		}
	}
	return false
}

func All(vs []UnitPrice, f func(UnitPrice) bool) bool {
	for _, v := range vs {
		if !f(v) {
			return false
		}
	}
	return true
}

func Filter(vs []UnitPrice, f func(UnitPrice) bool) []UnitPrice {
	vsf := make([]UnitPrice, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

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

func IntersectionForEach(vs1, vs2 []UnitPrice, t mapFunc) {
	for _, v := range vs1 {
		if i := Index(vs2, v); i >= 0 {
			t(v, vs2[i])
		}
	}
}
