package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/cryptix/wav"
)

var (
	inputFile              = flag.String("input", "", "input filename")
	outputFile             = flag.String("output", "", "output filename")
	lowFrequency           = flag.Float64("low_freq", 500.0, "lowest frequency of interest")
	highFrequency          = flag.Float64("high_freq", 5000.0, "highest frequency of interest")
	pixelsPerSecond        = flag.Float64("pixels_per_second", 1329, "pixels per second")
	imageHeight            = flag.Float64("image_height", 1000, "image height")
	freqMultiplier         = flag.Float64("freq_multiplier", 1.0, "frequency multiplier")
	speedMultiplier        = flag.Float64("speed_multiplier", 1.0, "speed multiplier")
	silenceSpeedMultiplier = flag.Float64("silence_speed_multiplier", 1.0, "silence speed multiplier")
	harmonics              = flag.Int("harmonics", 1, "number of harmonics in output")
	harmonicsFalloffPower  = flag.Float64("harmonics_falloff_power", 2, "power of amplitude falloff of harmonics")
	vibratoIntensity       = flag.Float64("vibrato_intensity", 0, "intensity of vibrato (multiplier for base frequency)")
	tremoloIntensity       = flag.Float64("tremolo_intensity", 0, "intensity of tremolo")
	vibratoFrequency       = flag.Float64("vibrato_frequency", 5, "frequency of vibrato")
	expectPixels           = flag.Bool("pixel_input", true, "expect input in the form of pixels")
)

type soundPoint struct {
	t    float64
	freq float64
}

type sound struct {
	begin  float64
	end    float64
	points []soundPoint
}

func (s soundPoint) String() string {
	return fmt.Sprintf("%0.2fs:%0.2fHz", s.t, s.freq)
}

func (s sound) String() string {
	var xs []string
	for _, x := range s.points {
		xs = append(xs, x.String())
	}
	return strings.Join(xs, "-")
}

type soundInst struct {
	playing    bool
	timeSpent  float64
	u          float64
	s          *sound
	env        *asdr
	tremoloMul float64
}

type asdr struct {
	attack, decay float64
	attackTime    float64
	decayTime     float64
	releaseTime   float64
	totalTime     float64
}

func (a *asdr) value(t float64) float64 {
	if t < a.attackTime {
		return t / a.attackTime * a.attack
	}

	if (a.totalTime - t) < a.releaseTime {
		return a.decay * (a.totalTime - t) / a.releaseTime
	}

	t -= a.attackTime

	if t < a.decayTime {
		return a.attack + t/a.decayTime*(a.decay-a.attack)
	}
	t -= a.decayTime

	return a.decay
}

func newASDR(totalTime float64) *asdr {
	return &asdr{
		attack:      1.0,
		decay:       0.8,
		attackTime:  0.05,
		decayTime:   0.05,
		releaseTime: 0.05,
		totalTime:   totalTime,
	}
}

func (s sound) play() *soundInst {
	return &soundInst{true, 0.0, 0.0, &s, newASDR(s.end - s.begin), 1.0}
}

func (s *soundInst) freq() float64 {
	t := s.timeSpent + s.s.begin
	for i := range s.s.points[:len(s.s.points)-1] {
		if s.s.points[i].t <= t && t <= s.s.points[i+1].t {
			rt := (t - s.s.points[i].t) / (s.s.points[i+1].t - s.s.points[i].t)
			return (s.s.points[i].freq + rt*(s.s.points[i+1].freq-s.s.points[i].freq))
		}
	}
	return s.s.points[0].freq
}

func (s *soundInst) value() float64 {
	if !s.playing {
		return 0.0
	}
	a := 0.15 * s.env.value(s.timeSpent) * s.tremoloMul
	var rv float64
	for i := 1; i <= *harmonics; i++ {
		rv += math.Sin(s.u*float64(i)) / math.Pow(float64(i), *harmonicsFalloffPower)
	}
	return a * rv
}

func (s *soundInst) addTime(dt float64) {
	s.timeSpent += dt
	if s.timeSpent >= (s.s.end - s.s.begin) {
		s.playing = false
		return
	}

	correction := *freqMultiplier / *speedMultiplier
	baseFreq := s.freq()
	freq := baseFreq
	if *vibratoIntensity > 0 || *tremoloIntensity > 0 {
		v := math.Sin(*vibratoFrequency * s.timeSpent * (math.Pi * 2.0))
		vibratoMul := *vibratoIntensity * v
		s.tremoloMul = 1.0 + *tremoloIntensity*v
		freq += vibratoMul * baseFreq
	}

	s.u += math.Pi * freq * dt * correction
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

			if componentno == 0 {
				snd.begin = secs
				snd.end = secs
			} else {
				if secs < snd.end {
					return nil, fmt.Errorf("line %d: component %q is moving backwards (%f < %f)", lineno, comp, secs, snd.end)
				}
			}
			snd.points = append(snd.points, soundPoint{
				t:    secs,
				freq: freq,
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
				rv[i].points[j].t -= minTime
			}
		}
	}

	return rv, nil
}

func playSounds(sounds []sound, step float64) <-chan float64 {
	ch := make(chan float64, 1024)
	go func() {
		t := 0.0
		var activeSounds []*soundInst
		already := map[int]bool{}

		for len(activeSounds) > 0 || len(already) < len(sounds) {
			for i, sound := range sounds {
				if already[i] {
					continue
				}
				if t >= sound.begin {
					already[i] = true
					activeSounds = append(activeSounds, sound.play())
				}
			}

			var v float64
			var nextActive []*soundInst
			for _, s := range activeSounds {
				v += s.value()
				s.addTime(step)
				if s.playing {
					nextActive = append(nextActive, s)
				}
			}
			activeSounds = nextActive

			ch <- v

			if len(activeSounds) == 0 {
				t += *silenceSpeedMultiplier * step
			} else {
				t += step
			}
		}

		close(ch)
	}()

	return ch
}

func writeWav(filename string, sampleRate int, ch <-chan float64) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	wavFile := &wav.File{
		SampleRate:      uint32(sampleRate),
		SignificantBits: 32,
		Channels:        1,
	}
	wr, err := wavFile.NewWriter(f)
	if err != nil {
		f.Close()
		return err
	}

	for x := range ch {
		val := int32(float64(2147483648) * x)
		if err := wr.WriteInt32(val); err != nil {
			f.Close()
			return err
		}
	}

	if err := wr.Close(); err != nil {
		return err
	}

	return nil
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

	step := *speedMultiplier / float64(sampleRate)

	if err := writeWav(*outputFile, sampleRate, playSounds(sounds, step)); err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	if err := mainCore(); err != nil {
		log.Fatalf("failure: %v", err)
	}
}
