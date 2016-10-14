package main

import (
	"errors"
	"flag"
	"log"

	"github.com/steinarvk/abora/synth/chirp"
	"github.com/steinarvk/abora/synth/envelope"
	"github.com/steinarvk/abora/synth/harmonics"
	"github.com/steinarvk/abora/synth/oscillator"
	"github.com/steinarvk/abora/synth/varying"
	"github.com/steinarvk/abora/wav"
)

var (
	frequency            = flag.Float64("freq", 440.0, "frequency (in Hz)")
	alternateFrequency   = flag.Float64("alternate_freq", 640.0, "alternate frequency (in Hz)")
	duration             = flag.Float64("duration", 2.0, "duration (in seconds)")
	outputFilename       = flag.String("output", "", "output filename")
	numberOfHarmonics    = flag.Int("harmonics", 0, "number of additional harmonics")
	harmonicsFalloff     = flag.Float64("harmonics_falloff", 2.0, "power of harmonics falloff")
	vibrato              = flag.Bool("vibrato", false, "use vibrato")
	vibratoFrequency     = flag.Float64("vibrato_frequency", 7.0, "vibrato frequency")
	vibratoAmplitude     = flag.Float64("vibrato_amplitude", 0.1, "vibrato amplitude (frequency multiplier)")
	amplitude            = flag.Float64("amplitude", 0.25, "amplitude multiplier")
	randomization        = flag.Int("randomization", 0, "randomization")
	randomizationWidth   = flag.Float64("randomization_width", 0.1, "randomization width multiplier")
	randomizationFalloff = flag.Float64("randomization_falloff", 5.0, "randomization falloff")
)

func mainCore() error {
	if *outputFilename == "" {
		return errors.New("--output is required")
	}
	osc := oscillator.Sin()

	if *randomization > 0 {
		osc = oscillator.Randomized(osc, *randomization, *randomizationWidth, oscillator.ExponentialFalloff(*randomizationFalloff))
	}
	osc = harmonics.WithHarmonics(osc,
		harmonics.SimpleSeq(*numberOfHarmonics+1, *harmonicsFalloff))

	env := envelope.LinearADSR(*duration, envelope.ADSRSpec{
		AttackDuration:  0.15,
		DecayDuration:   0.15,
		SustainLevel:    0.7,
		ReleaseDuration: 1.0,
	})
	env = envelope.Composite(env, envelope.Constant(*amplitude))

	freq := varying.Varying(varying.Constant(*frequency))
	if *vibrato {
		freq = varying.NewOscillating(
			freq,
			varying.OscillationFreq(varying.Constant(*vibratoFrequency)),
			varying.MultiplicativeOscillation(varying.Constant(*vibratoAmplitude)),
		)
	}

	sampleRate := 44110

	return wav.WriteFile(*outputFilename, sampleRate, chirp.New(freq, osc, env).AsChannel(sampleRate))
}

func main() {
	flag.Parse()

	if err := mainCore(); err != nil {
		log.Fatalf("failure: %v", err)
	}
}
