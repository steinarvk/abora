package envelope

import (
	"github.com/steinarvk/abora/synth/interpolation"
	"github.com/steinarvk/abora/synth/varying"
)

// Envelope represents the amplitude component of a waveform. It takes on
// values in [0,1]. It may or may not "end", i.e. reach zero permanently.
// Its argument is in seconds.
type Envelope interface {
	Amplitude() float64
	Done() bool
	Advance(float64)
}

type brickWall struct {
	limit float64
}

func BrickWall(t float64) Envelope {
	return &brickWall{t}
}

func (e *brickWall) Amplitude() float64 {
	if e.limit > 0 {
		return 1
	}
	return 0
}

func (e *brickWall) Done() bool {
	return e.limit <= 0
}

func (e *brickWall) Advance(dt float64) {
	e.limit -= dt
}

type Constant float64

func (x Constant) Amplitude() float64 { return float64(x) }
func (x Constant) Done() bool         { return float64(x) == 0 }
func (_ Constant) Advance(_ float64)  {}

var (
	Null     = Constant(0)
	Identity = Constant(1)
)

func sectionAttackSustain(attackDur, stabilizeDur float64, sustainLevel float64, interpolator interpolation.Function) Envelope {
	return &interpolatedEnvelope{amplitude: varying.NewInterpolated(
		[]varying.Point{
			{Time: 0, Value: 0},
			{Time: attackDur, Value: 1.0},
			{Time: attackDur + stabilizeDur, Value: sustainLevel},
		},
		varying.Interpolation(interpolator),
		varying.Infinite{},
	)}
}

func sectionRelease(beforeReleaseDur, releaseDur float64, interpol interpolation.Function) Envelope {
	vary := varying.NewInterpolated(
		[]varying.Point{
			{Time: 0, Value: 1},
			{Time: beforeReleaseDur, Value: 1},
			{Time: beforeReleaseDur + releaseDur, Value: 0},
		},
		varying.Interpolation(interpol),
	)
	return &interpolatedEnvelope{
		amplitude: vary,
		timeLeft:  beforeReleaseDur + releaseDur,
		finite:    true,
	}
}

type ADSRSpec struct {
	AttackDuration  float64
	DecayDuration   float64
	SustainLevel    float64
	ReleaseDuration float64
}

func LinearADSR(totalDuration float64, spec ADSRSpec) Envelope {
	return adsrWith(spec, totalDuration, interpolation.Linear)
}

func CosADSR(totalDuration float64, spec ADSRSpec) Envelope {
	return adsrWith(spec, totalDuration, interpolation.Cosine)
}

func adsrWith(spec ADSRSpec, totalDuration float64, interpol interpolation.Function) Envelope {
	beforeReleaseDur := totalDuration - spec.ReleaseDuration
	if beforeReleaseDur < 0 {
		beforeReleaseDur = 0
	}
	return Composite(
		sectionRelease(
			beforeReleaseDur,
			spec.ReleaseDuration,
			interpol),
		sectionAttackSustain(
			spec.AttackDuration,
			spec.DecayDuration,
			spec.SustainLevel,
			interpol),
	)
}

type interpolatedEnvelope struct {
	amplitude varying.Varying
	finite    bool
	timeLeft  float64
}

func (x *interpolatedEnvelope) Amplitude() float64 {
	if x.Done() {
		return 0.0
	}
	return x.amplitude.Value()
}

func (x *interpolatedEnvelope) Advance(dt float64) {
	x.amplitude.Advance(dt)
	if x.finite {
		x.timeLeft -= dt
	}
}

func (x *interpolatedEnvelope) Done() bool {
	return x.finite && x.timeLeft <= 0
}

type compositeEnvelope []Envelope

func Composite(components ...Envelope) Envelope {
	return compositeEnvelope(components)
}

func (c compositeEnvelope) Advance(dt float64) {
	for _, e := range c {
		e.Advance(dt)
	}
}

func (c compositeEnvelope) Done() bool {
	for _, e := range c {
		if e.Done() {
			return true
		}
	}
	return false
}

func (c compositeEnvelope) Amplitude() float64 {
	value := 1.0
	for _, e := range c {
		value *= e.Amplitude()
	}
	return value
}
