package chirp

import (
	"github.com/steinarvk/abora/synth/envelope"
	"github.com/steinarvk/abora/synth/oscillator"
	"github.com/steinarvk/abora/synth/varying"
)

type TremoloEnvelope struct {
	osc      oscillator.Oscillator
	strength float64
}

func (_ *TremoloEnvelope) Done() bool         { return false }
func (e *TremoloEnvelope) Advance(dt float64) { e.osc.Advance(dt) }
func (e *TremoloEnvelope) Amplitude() float64 {
	val := 0.5 * (e.osc.Value() + 1)
	remaining := 1.0 - e.strength*val
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

type Chirp interface {
	Sample() float64
	Done() bool
	Advance(float64)
}

type TimedChirp struct {
	Time  float64
	Chirp Chirp
}

func At(t float64, chirp Chirp) TimedChirp {
	return TimedChirp{
		Time:  t,
		Chirp: chirp,
	}
}

type chirp struct {
	osc     oscillator.Oscillator
	env     envelope.Envelope
	freq    varying.Varying
	tremolo varying.Varying
}

func (c *chirp) Sample() float64 {
	rv := c.osc.Value()
	rv *= c.env.Amplitude()
	if c.tremolo != nil {
		rv *= c.tremolo.Value()
	}
	return rv
}

func (c *chirp) Done() bool {
	return c.env.Done()
}

func (c *chirp) Advance(dt float64) {
	c.osc.Advance(c.freq.Value() * dt)
	c.env.Advance(dt)
	c.freq.Advance(dt)
	if c.tremolo != nil {
		c.tremolo.Advance(dt)
	}
}

func New(freq varying.Varying, osc oscillator.Oscillator, env envelope.Envelope) Chirp {
	return &chirp{
		osc:  osc,
		env:  env,
		freq: freq,
	}
}

// essential operations and how to achieve them:
//   - add harmonics => add an Oscillator
//   - control volume  => add a Constant() envelope.
//   - ADSR => envelope
//   - tremolo => envelope (containing an oscillator)
