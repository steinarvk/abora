package interpolation

import (
	"math"
)

type Function func(float64, float64, float64) float64

func Linear(t, x0, x1 float64) float64 {
	return x0 + t*(x1-x0)
}

func Cosine(t, x0, x1 float64) float64 {
	v := 1.0 - math.Cos(t*math.Pi*0.5)
	return Linear(v, x0, x1)
}
