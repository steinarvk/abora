package stats

import (
	"sort"
	"sync"
)

type ValueCollection struct {
	mutex  sync.Mutex
	values []float64
	sorted bool
}

func New() *ValueCollection {
	return &ValueCollection{}
}

func (v *ValueCollection) Max() float64 {
	return v.Quantile(1)
}

func (v *ValueCollection) Min() float64 {
	return v.Quantile(0)
}

func (v *ValueCollection) Add(x float64) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.values = append(v.values, x)
	v.sorted = false
}

func (v *ValueCollection) ReverseQuantile(x float64) float64 {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if !v.sorted {
		sort.Float64s(v.values)
		v.sorted = true
	}

	if x < v.values[0] {
		return 0.0
	}

	if x > v.values[len(v.values)-1] {
		return 1.0
	}

	i := sort.Search(len(v.values), func(i int) bool {
		return v.values[i] > x
	})

	return float64(i) / float64(len(v.values))
}

func (v *ValueCollection) Quantile(t float64) float64 {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if !v.sorted {
		sort.Float64s(v.values)
		v.sorted = true
	}
	if t <= 0 {
		return v.values[0]
	}
	if t >= 1 {
		return v.values[len(v.values)-1]
	}
	i := int(t * float64(len(v.values)))
	return v.values[i]
}
