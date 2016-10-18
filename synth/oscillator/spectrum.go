package oscillator

import (
	"log"
	"math"

	aborapb "github.com/steinarvk/abora/proto"
)

var (
	defaultSpectrumFreq = 440.0
)

func FromSpectrum(spec *aborapb.Spectrum) Oscillator {
	nominal := spec.NominalFrequency
	if nominal == 0.0 {
		nominal = defaultSpectrumFreq
	}

	// Oscillators should be tuned to 1Hz.
	// The correction should mean that when this oscillator
	// is played back at its nominal frequency, it sounds
	// like the original spectrum.
	correction := 1.0 / nominal

	log.Printf("loading spectrum: %v", spec)

	rv := &multiOscillator{}
	var totalW float64
	for _, p := range spec.Points {
		rv.w = append(rv.w, p.Amplitude)
		rv.mul = append(rv.mul, p.Frequency*correction)
		osc := Sin()
		if p.Phase > 0.0 {
			twoPiPhase := p.Phase
			unitPhase := (math.Pi + twoPiPhase) / (2 * math.Pi)
			osc.Advance(unitPhase)
		}
		rv.osc = append(rv.osc, osc)
		totalW += p.Amplitude
	}
	rv.multiMul = 1.0 / math.Sqrt(totalW)

	return rv
}
