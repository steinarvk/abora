package oscillator

import (
	"math"
)

// Oscillator represents the frequency component of a waveform. It takes on
// values in [-1,1] and will usually reach both extremes. It is stateful,
// containing its current phase.
type Oscillator interface {
	Value() float64
	Advance(float64)
	Clone() Oscillator
}

type Null struct{}

func (_ Null) Advance(_ float64) {}
func (_ Null) Value() float64    { return 0.0 }
func (_ Null) Clone() Oscillator { return Null{} }

func Sin() Oscillator { return &sinOsc{} }

type sinOsc struct {
	u float64
}

const (
	twoPi = 2 * math.Pi
)

func (s *sinOsc) Advance(du float64) {
	s.u += twoPi * du
}
func (s *sinOsc) Value() float64    { return math.Sin(s.u) }
func (s *sinOsc) Clone() Oscillator { return &sinOsc{s.u} }
