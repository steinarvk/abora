package main

import (
	"errors"
	"flag"
	"image"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/steinarvk/abora/colorscale"
	"github.com/steinarvk/abora/stats"

	"azul3d.org/engine/audio"
	_ "azul3d.org/engine/audio/flac"
	"github.com/mjibson/go-dsp/spectral"
)

var (
	inputFile         = flag.String("input", "", "input filename")
	windowSizeSeconds = flag.Float64("window_size_seconds", 0.01, "window size in seconds")
	pwelchNFFT        = flag.Int("pwelch_nfft", 8192, "PWelchOptions.NFFT")
	pwelchPad         = flag.Int("pwelch_pad", 8192, "PWelchOptions.Pad")
	lowFrequency      = flag.Float64("low_freq", 500.0, "lowest frequency of interest")
	highFrequency     = flag.Float64("high_freq", 5000.0, "highest frequency of interest")
	outputSpectrogram = flag.String("output_spectrogram", "", "output filename of spectrogram")
)

func readSine(freq float64, sampleRate int, secs float64) (int, <-chan float64, error) {
	ch := make(chan float64, 10000)

	timestep := 1.0 / float64(sampleRate)

	freqSlide := freq

	amplitude := 0.5

	go func() {
		t := 0.0
		for t < secs {
			realFreq := freq + t/secs*freqSlide
			value := amplitude * math.Sin(math.Pi*realFreq*t)
			ch <- value
			t += timestep
		}
		close(ch)
	}()

	return sampleRate, ch, nil
}

func readFile(filename string, lateErr *error) (int, <-chan float64, error) {
	*lateErr = nil

	fileHandle, err := os.Open(*inputFile)
	if err != nil {
		return 0, nil, err
	}

	decoder, format, err := audio.NewDecoder(fileHandle)
	if err != nil {
		fileHandle.Close()
		return 0, nil, err
	}

	config := decoder.Config()

	log.Printf("using decoder %q with config %v", format, config)

	if config.Channels != 1 {
		fileHandle.Close()
		return config.SampleRate, nil, errors.New("expected mono audio file")
	}

	seconds := 1
	bufsize := seconds * config.SampleRate
	underlying := audio.Float64{}
	buf := underlying.Make(bufsize, bufsize)

	ch := make(chan float64, bufsize)

	go func() {
		defer close(ch)
		defer fileHandle.Close()

		var total int64

		for {
			read, err := decoder.Read(buf)
			if err != nil && err != audio.EOS {
				*lateErr = err
				return
			}

			for i := 0; i < read; i++ {
				val := buf.At(i)
				ch <- val
				total++
			}

			if err == audio.EOS {
				break
			}
		}

		log.Printf("%d frames read", total)
	}()

	return config.SampleRate, ch, nil
}

type frequencyRange struct {
	fromFreq float64
	toFreq   float64
}

func (r frequencyRange) amountInside(f0, f1 float64) float64 {
	if r.fromFreq >= f1 {
		return 0.0
	}
	if r.toFreq <= f0 {
		return 0.0
	}
	if r.fromFreq > f0 {
		f0 = r.fromFreq
	}
	if r.toFreq < f1 {
		f1 = r.toFreq
	}
	return f1 - f0
}

func (r frequencyRange) density(pxx, freqs []float64) float64 {
	var total float64
	for i, w := range pxx[:len(pxx)-1] {
		f0 := freqs[i]
		f1 := freqs[i+1]
		total += r.amountInside(f0, f1) * w
	}
	return total
}

func onWindows(sz int, mod int, ch <-chan float64, f func(int64, []float64) error) error {
	n := 10
	bufsz := sz * n
	buf := make([]float64, bufsz)
	i := 0
	smplno := int64(0)

	for x := range ch {
		if i >= bufsz {
			copy(buf, buf[bufsz-sz:])
			i = sz
		}

		buf[i] = x
		smplno++
		i++

		if smplno >= int64(sz) {
			if smplno%int64(mod) != 0 {
				continue
			}
			if err := f(smplno, buf[i-sz:i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func findMostDenseFreq(pxx, freqs []float64) float64 {
	champion := 0

	for i, w := range pxx[1:] {
		if w > pxx[champion] {
			champion = i
		}
	}

	return freqs[champion]
}

func powerOfTwoAbove(x int) int {
	rv := 1
	for rv < x {
		rv *= 2
	}
	return rv
}

func rootMeanSquare(xs []float64) float64 {
	var rv float64
	for _, x := range xs {
		rv += x * x
	}
	return math.Sqrt(rv / float64(len(xs)))
}

func maxAbsFloat64(xs []float64) float64 {
	var v float64
	for _, x := range xs {
		x = math.Abs(x)
		if x > v {
			v = x
		}
	}
	return v
}

func mainCore() error {
	if *inputFile == "" {
		return errors.New("--input is required")
	}

	pwelchOpts := &spectral.PwelchOptions{
		NFFT:      *pwelchNFFT,
		Pad:       *pwelchPad,
		Scale_off: false,
	}

	var lateErr error

	sampleRate, ch, err := readFile(*inputFile, &lateErr)
	if err != nil {
		return err
	}

	// freqRange := int(*highFrequency - *lowFrequency)

	vcoll := stats.New()

	width := 0
	height := 1000
	var img *image.RGBA64
	firstScan := true
	index := 0

	var freqranges []frequencyRange

	freqStep := (*highFrequency - *lowFrequency) / float64(height)
	for i := height; i >= 0; i-- {
		fromFreq := *lowFrequency + float64(i)*freqStep
		toFreq := *lowFrequency + float64(i+1)*freqStep
		freqranges = append(freqranges, frequencyRange{fromFreq, toFreq})
	}

	sampleFreq := float64(sampleRate)
	windowSize := powerOfTwoAbove(int(sampleFreq * 0.05))
	everyNth := int(sampleFreq * 0.00075)

	log.Printf("sampleFreq %f windowSize %d", sampleFreq, windowSize)

	pwelchOpts.NFFT = windowSize

	log.Printf("using window size %v and options %v", windowSize, pwelchOpts)

	f := func(sampleNo int64, xs []float64) error {

		t := float64(sampleNo) / float64(sampleFreq)
		pxx, freqs := spectral.Pwelch(xs, sampleFreq, pwelchOpts)
		ampl := maxAbsFloat64(xs)
		rms := rootMeanSquare(xs)

		log.Printf("processing at %f ampl %f rms %f", t, ampl, rms)
		/*
			var totalW float64
			for i, w := range pxx {
				totalW += w
				log.Printf("  at %f:\thz=%f\tw=%f\tcumw=%f", t, freqs[i], w, totalW)
			}
		*/

		for j, fr := range freqranges {
			d := fr.density(pxx, freqs)
			if firstScan {
				vcoll.Add(d)
			} else {
				nearmax := vcoll.Quantile(0.995) * 0.99
				threshold := 0.5 * (vcoll.Quantile(0.5) + nearmax)
				threshold = vcoll.Quantile(0.8)
				t := 0.0
				if d > threshold {
					t = (math.Log(d) - math.Log(threshold)) / (math.Log(nearmax) - math.Log(threshold))
				}
				img.Set(index, j, colorscale.Viridis(t))
			}
		}

		if firstScan {
			width++
		} else {
			index++
		}

		return nil
	}

	if err := onWindows(windowSize, everyNth, ch, f); err != nil {
		return err
	}

	if lateErr != nil {
		return lateErr
	}

	img = image.NewRGBA64(image.Rect(0, 0, width, height))
	firstScan = false
	sampleRate, ch, err = readFile(*inputFile, &lateErr)
	if err != nil {
		return err
	}
	if err := onWindows(windowSize, everyNth, ch, f); err != nil {
		return err
	}

	if *outputSpectrogram != "" {
		f, err := os.Create(*outputSpectrogram)
		if err != nil {
			return err
		}
		defer f.Close()

		if err := png.Encode(f, img); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Parse()

	if err := mainCore(); err != nil {
		log.Fatalf("failure: %v", err)
	}
}
