package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/steinarvk/abora/synth/chirp"
	"github.com/steinarvk/abora/synth/envelope"
	"github.com/steinarvk/abora/synth/harmonics"
	"github.com/steinarvk/abora/synth/interpolation"
	"github.com/steinarvk/abora/synth/mix"
	"github.com/steinarvk/abora/synth/oscillator"
	"github.com/steinarvk/abora/synth/varying"
	"github.com/steinarvk/abora/wav"
)

var (
	inputFile       = flag.String("input", "", "input filename")
	outputFile      = flag.String("output", "", "output filename")
	pixelsPerSecond = flag.Float64("pixels_per_second", 1329, "pixels per second")
	imageHeight     = flag.Float64("image_height", 1000, "image height")
	expectPixels    = flag.Bool("pixel_input", true, "expect input in the form of pixels")
	flattenWithin   = flag.Float64("flatten_within", 0, "flatten frequency changes within")
	lowFrequency    = flag.Float64("low_freq", 500.0, "lowest frequency of interest")
	highFrequency   = flag.Float64("high_freq", 5000.0, "highest frequency of interest")

	amplitude         = flag.Float64("amplitude", 0.25, "amplitude")
	freqMultiplier    = flag.Float64("freq_multiplier", 1.0, "frequency multiplier")
	timeMultiplier    = flag.Float64("time_multiplier", 1.0, "time multiplier")
	numberOfHarmonics = flag.Int("harmonics", 0, "number of additional harmonics")
	harmonicsFalloff  = flag.Float64("harmonics_falloff", 2.0, "power of harmonics falloff")
	vibratoIntensity  = flag.Float64("vibrato_intensity", 0.0, "intensity of vibrato")
	vibratoFrequency  = flag.Float64("vibrato_frequency", 7.0, "vibrato frequency")
	tremoloIntensity  = flag.Float64("tremolo_intensity", 0.0, "intensity of vibrato")
	tremoloFrequency  = flag.Float64("tremolo_frequency", 7.0, "vibrato frequency")
)

type sound struct {
	begin  float64
	end    float64
	points []varying.Point
}

func (s sound) String() string {
	var xs []string
	for _, x := range s.points {
		xs = append(xs, fmt.Sprintf("%0.2fs:%0.2fHz", x.Time, x.Value))
	}
	return strings.Join(xs, "-")
}

func (s sound) asChirp() chirp.TimedChirp {
	duration := (s.end - s.begin) * *timeMultiplier
	osc := oscillator.Sin()

	log.Printf("total duration %v", duration)

	env := envelope.LinearADSR(duration, envelope.ADSRSpec{
		AttackDuration:  0.05,
		DecayDuration:   0.05,
		SustainLevel:    0.7,
		ReleaseDuration: 0.05,
	})
	env = envelope.Composite(env, envelope.Constant(*amplitude))
	freq := varying.NewInterpolated(s.points, varying.Interpolation(interpolation.Cosine))

	if *numberOfHarmonics > 0 {
		osc = harmonics.WithHarmonics(osc,
			harmonics.SimpleSeq(*numberOfHarmonics+1, *harmonicsFalloff))
	}

	if *vibratoIntensity > 0.0 {
		freq = varying.NewOscillating(
			freq,
			varying.OscillationFreq(varying.Constant(*vibratoFrequency)),
			varying.MultiplicativeOscillation(varying.Constant(*vibratoIntensity)),
		)
	}

	if *tremoloIntensity > 0.0 {
		log.Fatalf("TODO: tremolo")
	}

	freq = varying.Map(freq, func(t float64) float64 {
		return t * *freqMultiplier
	})

	log.Printf("this chirp is timed for %v", s.begin)

	return chirp.TimedChirp{
		Time:  s.begin * *timeMultiplier,
		Chirp: chirp.New(freq, osc, env),
	}
}

func parseAdHocFormat(filename string) ([]sound, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rv []sound

	scanner := bufio.NewScanner(f)
	lineno := 0
	for scanner.Scan() {
		lineno++

		t := strings.TrimSpace(scanner.Text())
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}

		components := strings.Split(t, "-")
		if len(components) < 2 {
			return nil, fmt.Errorf("line %d: expected more than one component", lineno)
		}

		var snd sound

		var lastFreq *float64

		for componentno, comp := range components {
			xAndY := strings.Split(comp, ":")
			if len(xAndY) != 2 {
				return nil, fmt.Errorf("line %d: component %q invalid", lineno, comp)
			}
			x, err := strconv.ParseFloat(xAndY[0], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: component %q invalid (%v)", lineno, comp, err)
			}
			y, err := strconv.ParseFloat(xAndY[1], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: component %q invalid (%v)", lineno, comp, err)
			}
			var secs, freq float64
			if *expectPixels {
				secs = float64(x) / *pixelsPerSecond
				freq = *highFrequency + (*lowFrequency-*highFrequency)*(float64(y)/float64(*imageHeight))
			} else {
				secs = float64(x)
				freq = float64(y)
			}

			if secs < 0 || secs > 3600 {
				return nil, fmt.Errorf("line %d: component %q has non-sensible time (%f)", lineno, comp, secs)
			}
			if freq < *lowFrequency || freq > *highFrequency {
				return nil, fmt.Errorf("line %d: component %q has non-sensible frequency (%f)", lineno, comp, freq)
			}

			if *flattenWithin > 0 {
				if lastFreq != nil && (freq-*lastFreq) < *flattenWithin {
					freq = *lastFreq
				} else {
					lastFreq = &freq
				}
			}

			if componentno == 0 {
				snd.begin = secs
				snd.end = secs
			} else {
				if secs < snd.end {
					return nil, fmt.Errorf("line %d: component %q is moving backwards (%f < %f)", lineno, comp, secs, snd.end)
				}
			}
			snd.points = append(snd.points, varying.Point{
				Time:  secs,
				Value: freq,
			})
			snd.end = secs
		}

		rv = append(rv, snd)
	}

	minTime := rv[0].begin

	for _, snd := range rv[1:] {
		if snd.begin < minTime {
			minTime = snd.begin
		}
	}

	log.Printf("minTime = %v", minTime)

	if minTime > 0 {
		for i, snd := range rv {
			rv[i].begin -= minTime
			rv[i].end -= minTime
			for j, _ := range snd.points {
				rv[i].points[j].Time -= minTime
			}
		}
	}

	return rv, nil
}

func playSounds(sounds []sound, sampleRate int) <-chan float64 {
	var tc []chirp.TimedChirp
	for _, snd := range sounds {
		tc = append(tc, snd.asChirp())
	}
	return mix.AsChannel(tc, sampleRate, 0.0)
}

func mainCore() error {
	sounds, err := parseAdHocFormat(*inputFile)
	if err != nil {
		return err
	}

	for _, sound := range sounds {
		fmt.Println(sound.String())
	}

	sampleRate := 44100

	return wav.WriteFile(*outputFile, sampleRate, playSounds(sounds, sampleRate))
}

func main() {
	flag.Parse()

	if err := mainCore(); err != nil {
		log.Fatalf("failure: %v", err)
	}
}
