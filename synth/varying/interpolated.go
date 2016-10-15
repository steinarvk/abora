package varying

import (
	"github.com/steinarvk/abora/synth/interpolation"
)

type Point struct {
	Time  float64
	Value float64
}

type interpolatedVarying struct {
	points       []Point
	interpolator interpolation.Function
	cyclic       bool
	infinite     bool
	t            float64
	index        int
}

type interpolatedOption interface {
	Apply(*interpolatedVarying)
}

type Interpolation interpolation.Function

func (f Interpolation) Apply(rv *interpolatedVarying) { rv.interpolator = interpolation.Function(f) }

type Cyclic struct{}

func (_ Cyclic) Apply(rv *interpolatedVarying) {
	rv.cyclic = true
}

type Infinite struct{}

func (_ Infinite) Apply(rv *interpolatedVarying) { rv.infinite = true }

func normalizePoints(points []Point) []Point {
	if len(points) < 1 {
		return nil
	}
	t0 := points[0].Time
	var rv []Point
	for _, p := range points {
		rv = append(rv, Point{Time: p.Time - t0, Value: p.Value})
	}
	return rv
}

func NewInterpolated(points []Point, opts ...interpolatedOption) Varying {
	rv := &interpolatedVarying{
		points:       normalizePoints(points),
		interpolator: interpolation.Linear,
		infinite:     true,
	}
	for _, opt := range opts {
		opt.Apply(rv)
	}
	return rv
}

func (e *interpolatedVarying) Value() float64 {
	n := len(e.points)
	if n == 0 {
		return 0.0
	}
	lastTime := e.points[n-1].Time
	if e.t >= lastTime {
		if !e.infinite {
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

func (e *interpolatedVarying) Advance(dt float64) {
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
			if !e.infinite {
				return
			}
			e.index = 0
			e.t -= lastTime
		} else {
			return
		}
	}
}
