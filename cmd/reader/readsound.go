package main

import (
	"errors"
	"flag"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/steinarvk/abora/analysis"
	"github.com/steinarvk/abora/colorscale"
	"github.com/steinarvk/abora/snippet"
)

var (
	inputFile         = flag.String("input", "", "input filename")
	windowSizeSeconds = flag.Float64("window_size_seconds", 0.05, "window size in seconds")
	pwelchNFFT        = flag.Int("pwelch_nfft", 8192, "PWelchOptions.NFFT")
	pwelchPad         = flag.Int("pwelch_pad", 8192, "PWelchOptions.Pad")
	lowFrequency      = flag.Float64("low_freq", 500.0, "lowest frequency of interest")
	highFrequency     = flag.Float64("high_freq", 5000.0, "highest frequency of interest")
	outputSpectrogram = flag.String("output_spectrogram", "", "output filename of spectrogram")
)

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

	log.Printf("reading input file %q", *inputFile)
	snip, err := snippet.Read(*inputFile)
	if err != nil {
		return err
	}

	log.Printf("analyzing")
	anal, err := analysis.Analyze(snip, &analysis.Params{
		MinWindowSizeSeconds:     *windowSizeSeconds,
		PwelchPadding:            pwelchPad,
		NumberOfFrequencyBuckets: 1000,
		Range: &analysis.FrequencyRange{
			LowHz:  *lowFrequency,
			HighHz: *highFrequency,
		},
		AnalysesPerSecond: 1.0 / 0.00075,
	})
	if err != nil {
		return err
	}

	log.Printf("visualizing")
	img := anal.Visualize(anal.DefaultValueMapper(), colorscale.Viridis)

	log.Printf("saving spectrogram")
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
