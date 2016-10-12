package envelope

import (
	"math"
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

type Point struct {
	Time  float64
	Value float64
}

func linearInterpolation(t, x0, x1 float64) float64 {
	return x0 + t*(x1-x0)
}

func cosInterpolation(t, x0, x1 float64) float64 {
	v := 1.0 - math.Cos(t*math.Pi*0.5)
	return linearInterpolation(v, x0, x1)
}

type Interpolator func(float64, float64, float64) float64

type interpolatedEnvelope struct {
	points       []Point
	interpolator Interpolator
	cyclic       bool
	finite       bool
	t            float64
	index        int
}

func sectionAttackSustain(attackDur, stabilizeDur float64, sustainLevel float64, interpolator Interpolator) Envelope {
	return &interpolatedEnvelope{
		points: []Point{
			{Time: 0, Value: 0},
			{Time: attackDur, Value: 1.0},
			{Time: attackDur + stabilizeDur, Value: sustainLevel},
		},
		interpolator: Interpolator(interpolator),
		cyclic:       false,
		finite:       false,
	}
}

func sectionRelease(beforeReleaseDur, releaseDur float64, interpol Interpolator) Envelope {
	return &interpolatedEnvelope{
		points: []Point{
			{Time: 0, Value: 1},
			{Time: beforeReleaseDur, Value: 1},
			{Time: beforeReleaseDur + releaseDur, Value: 0},
		},
		interpolator: Interpolator(interpol),
		cyclic:       false,
		finite:       true,
	}
}

type ADSRSpec struct {
	AttackDuration  float64
	DecayDuration   float64
	SustainLevel    float64
	ReleaseDuration float64
}

func LinearADSR(totalDuration float64, spec ADSRSpec) Envelope {
	return adsrWith(spec, totalDuration, linearInterpolation)
}

func CosADSR(totalDuration float64, spec ADSRSpec) Envelope {
	return adsrWith(spec, totalDuration, cosInterpolation)
}

func adsrWith(spec ADSRSpec, totalDuration float64, interpol Interpolator) Envelope {
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

func (e *interpolatedEnvelope) Amplitude() float64 {
	n := len(e.points)
	if n == 0 {
		return 0.0
	}
	lastTime := e.points[n-1].Time
	if e.t >= lastTime {
		if e.finite {
			return 0.0
		}
		return e.points[n-1].Value
	}
	v0 := e.points[e.index].Value
	v1 := e.points[e.index+1].Value
	if v0 == v1 {
		return v0
	}
	t0 := e.points[e.index].Time
	t1 := e.points[e.index+1].Time
	return e.interpolator((e.t-t0)/(t1-t0), v0, v1)
}

func (e *interpolatedEnvelope) Advance(dt float64) {
	n := len(e.points)
	if n == 0 {
		return
	}
	lastTime := e.points[n-1].Time
	e.t += dt
	if e.t >= lastTime && !e.cyclic {
		return
	}
	for {
		for (e.index+1) < n && e.t >= e.points[e.index+1].Time {
			e.index++
		}
		if (e.index + 1) >= n {
			if e.finite {
				return
			}
			e.index = 0
			e.t -= lastTime
		} else {
			return
		}
	}
}

func (e *interpolatedEnvelope) Done() bool {
	n := len(e.points)
	if e.cyclic || !e.finite {
		return false
	}
	if n < 1 {
		return false
	}
	return e.t > e.points[n-1].Time
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
