package oscillator

import (
	"math"
	"math/rand"
)

type multiOscillator struct {
	multiMul float64
	w        []float64
	mul      []float64
	osc      []Oscillator
}

func (m *multiOscillator) Value() float64 {
	var rv float64
	for i, osc := range m.osc {
		rv += osc.Value() * m.w[i]
	}
	return rv * m.multiMul
}

func (m *multiOscillator) Advance(dt float64) {
	for i, osc := range m.osc {
		osc.Advance(m.mul[i] * dt)
	}
}

func (m *multiOscillator) Clone() Oscillator {
	rv := &multiOscillator{multiMul: m.multiMul}
	for i, osc := range m.osc {
		rv.w = append(rv.w, m.w[i])
		rv.mul = append(rv.mul, m.mul[i])
		rv.osc = append(rv.osc, osc.Clone())
	}
	return rv
}

func LinearFalloff(t float64) float64 {
	return t
}

func ExponentialFalloff(c float64) func(float64) float64 {
	return func(t float64) float64 {
		return math.Exp(-c * t)
	}
}

type WeightingFunction func(float64) float64

func Randomized(osc Oscillator, n int, width float64, weight WeightingFunction) Oscillator {
	rv := &multiOscillator{}
	totalW := 0.0
	for i := 0; i < n; i++ {
		p := rand.Float64()
		w := weight(1.0 - p)
		totalW += w
		mul := 1 + width*(p*2-1)
		osc := osc.Clone()
		osc.Advance(rand.Float64())
		rv.w = append(rv.w, w)
		rv.mul = append(rv.mul, mul)
		rv.osc = append(rv.osc, osc)
	}
	rv.multiMul = 1.0 / math.Sqrt(totalW)
	return rv
}
