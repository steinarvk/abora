package main

import (
	"errors"
	"flag"
	"log"

	"github.com/steinarvk/abora/synth/chirp"
	"github.com/steinarvk/abora/synth/envelope"
	"github.com/steinarvk/abora/synth/oscillator"
	"github.com/steinarvk/abora/wav"
)

var (
	frequency      = flag.Float64("freq", 440.0, "frequency (in Hz)")
	duration       = flag.Float64("duration", 2.0, "duration (in seconds)")
	outputFilename = flag.String("output", "", "output filename")
)

func mainCore() error {
	if *outputFilename == "" {
		return errors.New("--output is required")
	}
	osc := oscillator.Sin()

	env := envelope.LinearADSR(*duration, envelope.ADSRSpec{
		AttackDuration:  0.15,
		DecayDuration:   0.15,
		SustainLevel:    0.7,
		ReleaseDuration: 1.0,
	})
	env = envelope.Composite(env, envelope.Constant(0.25))

	sampleRate := 44110

	return wav.WriteFile(*outputFilename, sampleRate, chirp.New(*frequency, osc, env).AsChannel(sampleRate))
}

func main() {
	flag.Parse()

	if err := mainCore(); err != nil {
		log.Fatalf("failure: %v", err)
	}
}
