package varying

import (
	"github.com/steinarvk/abora/synth/oscillator"
)

var (
	defaultOscillationHz = 5.0
)

type oscillatingVarying struct {
	val            Varying
	osc            oscillator.Oscillator
	freq           Varying
	additive       Varying
	multiplicative Varying
}

func (x *oscillatingVarying) Advance(dt float64) {
	x.osc.Advance(x.freq.Value() * dt)
	Advance(dt, x.val, x.osc, x.freq, x.additive, x.multiplicative)
}

func (x *oscillatingVarying) Value() float64 {
	base := x.val.Value()

	oscillation := x.osc.Value()

	var delta float64
	if x.additive != nil {
		delta += x.additive.Value()
	}
	if x.multiplicative != nil {
		delta += base * x.multiplicative.Value()
	}

	return base + delta*oscillation
}

type oscillatingOption interface {
	Apply(*oscillatingVarying)
}

type oscillationFreq struct{ v Varying }

func (o oscillationFreq) Apply(x *oscillatingVarying) { x.freq = Varying(o.v) }
func OscillationFreq(v Varying) oscillatingOption     { return oscillationFreq{v} }

type additiveOscillation struct{ v Varying }

func (o additiveOscillation) Apply(x *oscillatingVarying) { x.additive = Varying(o.v) }
func AdditiveOscillation(v Varying) oscillatingOption     { return additiveOscillation{v} }

type multiplicativeOscillation struct{ v Varying }

func (o multiplicativeOscillation) Apply(x *oscillatingVarying) { x.multiplicative = Varying(o.v) }
func MultiplicativeOscillation(v Varying) oscillatingOption     { return multiplicativeOscillation{v} }

func NewOscillating(val Varying, opts ...oscillatingOption) Varying {
	rv := &oscillatingVarying{
		val:  val,
		osc:  oscillator.Sin(),
		freq: Constant(defaultOscillationHz),
	}
	for _, opt := range opts {
		opt.Apply(rv)
	}
	return rv
}
