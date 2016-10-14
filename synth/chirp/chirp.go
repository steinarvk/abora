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

type Chirp struct {
	osc     oscillator.Oscillator
	env     envelope.Envelope
	freq    varying.Varying
	tremolo varying.Varying
}

func (c *Chirp) Sample() float64 {
	rv := c.osc.Value()
	rv *= c.env.Amplitude()
	if c.tremolo != nil {
		rv *= c.tremolo.Value()
	}
	return rv
}

func (c *Chirp) Done() bool {
	return c.env.Done()
}

func (c *Chirp) Advance(dt float64) {
	c.osc.Advance(c.freq.Value() * dt)
	c.env.Advance(dt)
	c.freq.Advance(dt)
	if c.tremolo != nil {
		c.tremolo.Advance(dt)
	}
}

func (c *Chirp) AsChannel(sampleRate int) <-chan float64 {
	bufsz := sampleRate
	ch := make(chan float64, bufsz)
	step := 1.0 / float64(sampleRate)
	go func() {
		for !c.Done() {
			c.Advance(step)
			ch <- c.Sample()
		}
		close(ch)
	}()
	return ch
}

func New(freq varying.Varying, osc oscillator.Oscillator, env envelope.Envelope) *Chirp {
	return &Chirp{
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
