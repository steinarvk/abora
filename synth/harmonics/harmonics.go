package harmonics

import (
	"math"

	"github.com/steinarvk/abora/synth/oscillator"
	"github.com/steinarvk/abora/synth/varying"
)

type Harmonic struct {
	FreqMul varying.Varying
	AmpMul  varying.Varying
}

func simpleHarmonic(freqMul, ampMul float64) Harmonic {
	return Harmonic{
		FreqMul: varying.Constant(freqMul),
		AmpMul:  varying.Constant(ampMul),
	}
}

func SimpleSeq(n int, power float64) []Harmonic {
	var rv []Harmonic
	for i := 1; i <= n; i++ {
		amp := 1.0 / math.Pow(float64(i), power)
		rv = append(rv, simpleHarmonic(float64(i), amp))
	}
	return rv
}

type withHarmonics struct {
	osc  []oscillator.Oscillator
	harm []Harmonic
}

func (h *withHarmonics) Clone() oscillator.Oscillator {
	rv := &withHarmonics{
		harm: h.harm,
	}
	for _, osc := range h.osc {
		rv.osc = append(rv.osc, osc.Clone())
	}
	return rv
}

func (h *withHarmonics) Value() float64 {
	var rv float64
	for i, harm := range h.harm {
		val := h.osc[i].Value()
		amp := harm.AmpMul.Value()
		rv += val * amp
	}
	return rv
}

func (h *withHarmonics) Advance(dt float64) {
	for i, harm := range h.harm {
		h.osc[i].Advance(harm.FreqMul.Value() * dt)
		harm.AmpMul.Advance(dt)
		harm.FreqMul.Advance(dt)
	}
}

func WithHarmonics(osc oscillator.Oscillator, harm []Harmonic) oscillator.Oscillator {
	rv := &withHarmonics{
		harm: harm,
	}
	for _ = range harm {
		rv.osc = append(rv.osc, osc.Clone())
	}
	return rv
}
