package chirp

import (
	"fmt"

	"github.com/steinarvk/abora/synth/envelope"
	"github.com/steinarvk/abora/synth/oscillator"
	"github.com/steinarvk/abora/synth/varying"

	pb "github.com/steinarvk/abora/proto"
)

var (
	defaultsContext = &pb.Context{
		Initial: &pb.PointSettings{
			Freq: &pb.DoubleOrHold{
				ValueOrHold: &pb.DoubleOrHold_Value{
					Value: 440.0,
				},
			},
			Amplitude: &pb.DoubleOrHold{
				ValueOrHold: &pb.DoubleOrHold_Value{
					Value: 1.0,
				},
			},
			TremoloStrength: &pb.DoubleOrHold{
				ValueOrHold: &pb.DoubleOrHold_Value{
					Value: 0,
				},
			},
			VibratoStrength: &pb.DoubleOrHold{
				ValueOrHold: &pb.DoubleOrHold_Value{
					Value: 0,
				},
			},
			TremoloFreq: &pb.DoubleOrHold{
				ValueOrHold: &pb.DoubleOrHold_Value{
					Value: 6,
				},
			},
			VibratoFreq: &pb.DoubleOrHold{
				ValueOrHold: &pb.DoubleOrHold_Value{
					Value: 6,
				},
			},
		},
		Oscillator: &pb.Oscillator{
			Oscillators: &pb.Oscillator_Sine{},
		},
		Envelope: &pb.Envelope{
			EnvelopeKind: &pb.Envelope_Adsr{
				Adsr: &pb.ADSREnvelope{
					AttackDuration:  0.15,
					DecayDuration:   0.15,
					SustainLevel:    0.7,
					ReleaseDuration: 0.5,
				},
			},
		},
	}
)

func OverrideContext(context, override *pb.Context) *pb.Context {
	if context == nil {
		context = &pb.Context{}
	}

	if override == nil {
		override = &pb.Context{}
	}

	if override.Oscillator != nil {
		context.Oscillator = override.Oscillator
	}

	if context.Initial == nil {
		context.Initial = &pb.PointSettings{}
	}

	if override.Envelope != nil {
		context.Envelope = override.Envelope
	}

	if override.Initial != nil && override.Initial.Freq != nil {
		context.Initial.Freq = override.Initial.Freq
	}

	if override.Initial != nil && override.Initial.Amplitude != nil {
		context.Initial.Amplitude = override.Initial.Amplitude
	}

	if override.Initial != nil && override.Initial.TremoloStrength != nil {
		context.Initial.TremoloStrength = override.Initial.TremoloStrength
	}

	if override.Initial != nil && override.Initial.TremoloFreq != nil {
		context.Initial.TremoloFreq = override.Initial.TremoloFreq
	}

	if override.Initial != nil && override.Initial.VibratoStrength != nil {
		context.Initial.VibratoStrength = override.Initial.VibratoStrength
	}

	if override.Initial != nil && override.Initial.VibratoFreq != nil {
		context.Initial.VibratoFreq = override.Initial.VibratoFreq
	}

	return context
}

func maybeAdd(xs []*pb.Point, x *pb.Point) []*pb.Point {
	if x != nil {
		xs = append(xs, x)
	}
	return xs
}

func makeVarying(def *pb.Point, xs []*pb.Point, name string, errOut *error, extractor func(*pb.PointSettings) *pb.DoubleOrHold) varying.Varying {
	if *errOut != nil {
		return nil
	}

	switch {
	case len(xs) == 0:
		xs = append(xs, def)
	case xs[0].T > 0:
		xs = append([]*pb.Point{def}, xs...)
	}

	if xs[0].T > 0 {
		*errOut = fmt.Errorf("interpolation sequence for %v: first value cannot have time > 0", name)
		return nil
	}

	extract := func(p *pb.Point, lastValue *float64) (float64, bool) {
		fail := func() (float64, bool) {
			*errOut = fmt.Errorf("interpolation sequence for %v: extraction failed (cannot extract from %v)", name, p)
			return 0, false
		}
		if p.Settings == nil {
			return fail()
		}
		val := extractor(p.Settings)
		if val == nil {
			return fail()
		}
		x, ok := val.GetValueOrHold().(*pb.DoubleOrHold_Value)
		if !ok {
			if val.GetHold() && lastValue != nil {
				return *lastValue, true
			}
			return fail()
		}
		return x.Value, true
	}

	lastValue, ok := extract(xs[0], nil)
	if !ok {
		return nil
	}

	var lastTime float64

	pts := []varying.Point{}

	for i, p := range xs {
		if i > 0 && lastTime >= p.T {
			*errOut = fmt.Errorf("interpolation sequence for %v: times not strictly ascending (%v >= %v)", name, lastTime, p.T)
			return nil
		}

		value, ok := extract(p, &lastValue)
		if !ok {
			return nil
		}

		pts = append(pts, varying.Point{
			Time:  p.T,
			Value: value,
		})

		lastTime = p.T
	}

	return varying.NewInterpolated(pts)
}

func FromProto(spec *pb.Chirp, context *pb.Context) (*TimedChirp, error) {
	context = OverrideContext(
		OverrideContext(
			OverrideContext(nil, defaultsContext),
			context),
		spec.ContextOverride)

	initialPoint := &pb.Point{T: 0, Settings: context.Initial}

	var freqDH, ampDH, tremStrDH, tremFreqDH, vibStrDH, vibFreqDH []*pb.Point

	for _, point := range spec.Points {
		set := point.Settings
		if set == nil {
			continue
		}
		if set.Freq != nil {
			freqDH = maybeAdd(freqDH, point)
		}
		if set.Amplitude != nil {
			ampDH = maybeAdd(ampDH, point)
		}
		if set.TremoloStrength != nil {
			tremStrDH = maybeAdd(tremStrDH, point)
		}
		if set.TremoloFreq != nil {
			tremFreqDH = maybeAdd(tremFreqDH, point)
		}
		if set.VibratoStrength != nil {
			vibStrDH = maybeAdd(vibStrDH, point)
		}
		if set.VibratoFreq != nil {
			vibFreqDH = maybeAdd(vibFreqDH, point)
		}
	}

	var err error

	freqV := makeVarying(initialPoint, freqDH, "Freq", &err, func(s *pb.PointSettings) *pb.DoubleOrHold {
		return s.GetFreq()
	})
	ampV := makeVarying(initialPoint, ampDH, "Amplitude", &err, func(s *pb.PointSettings) *pb.DoubleOrHold {
		return s.GetAmplitude()
	})
	tremStrV := makeVarying(initialPoint, tremStrDH, "TremoloStrength", &err, func(s *pb.PointSettings) *pb.DoubleOrHold {
		return s.GetTremoloStrength()
	})
	tremFreqV := makeVarying(initialPoint, tremFreqDH, "TremoloFreq", &err, func(s *pb.PointSettings) *pb.DoubleOrHold {
		return s.GetTremoloFreq()
	})
	vibStrV := makeVarying(initialPoint, vibStrDH, "VibratoStrength", &err, func(s *pb.PointSettings) *pb.DoubleOrHold {
		return s.GetVibratoStrength()
	})
	vibFreqV := makeVarying(initialPoint, vibFreqDH, "VibratoFreq", &err, func(s *pb.PointSettings) *pb.DoubleOrHold {
		return s.GetVibratoFreq()
	})

	if err != nil {
		return nil, err
	}

	env, err := envelope.FromProto(context.Envelope, spec.Duration)
	if err != nil {
		return nil, fmt.Errorf("constructing envelope from %v of duration %v: %v", context.Envelope, err)
	}

	osc, err := oscillator.FromProto(context.Oscillator)
	if err != nil {
		return nil, fmt.Errorf("constructing oscillator from %v: %v", context.Oscillator, err)
	}

	modifiedFreqV := varying.NewOscillating(freqV,
		varying.OscillationFreq(vibFreqV),
		varying.MultiplicativeOscillation(vibStrV))

	tremoloV := varying.NewOscillating(varying.Constant(1),
		varying.OscillationFreq(tremFreqV),
		varying.AdditiveOscillation(tremStrV))

	realEnv := envelope.WithVarying(env, ampV, tremoloV)

	rv := New(modifiedFreqV, osc, realEnv)

	return &TimedChirp{
		Time:  spec.BeginTime,
		Chirp: rv,
	}, nil
}
