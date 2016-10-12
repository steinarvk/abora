package chirp

import (
	"github.com/steinarvk/abora/synth/envelope"
	"github.com/steinarvk/abora/synth/oscillator"
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
	osc      oscillator.Oscillator
	env      envelope.Envelope
	baseFreq float64
}

func (c *Chirp) Sample() float64 {
	return c.osc.Value() * c.env.Amplitude()
}

func (c *Chirp) Done() bool {
	return c.env.Done()
}

func (c *Chirp) freq() float64 {
	return c.baseFreq
}

func (c *Chirp) Advance(dt float64) {
	c.osc.Advance(c.freq() * dt)
	c.env.Advance(dt)
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

func New(freq float64, osc oscillator.Oscillator, env envelope.Envelope) *Chirp {
	return &Chirp{osc, env, freq}
}

// essential operations and how to achieve them:
//   - add harmonics => add an Oscillator
//   - control volume  => add a Constant() envelope.
//   - ADSR => envelope
//   - tremolo => envelope (containing an oscillator)
