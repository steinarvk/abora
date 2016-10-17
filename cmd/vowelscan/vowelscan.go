package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"sort"

	"github.com/golang/protobuf/proto"

	"github.com/steinarvk/abora/analysis"
	"github.com/steinarvk/abora/snippet"
	"github.com/steinarvk/abora/stats"

	aborapb "github.com/steinarvk/abora/proto"
)

var (
	inputFile         = flag.String("input", "", "input filename")
	beginSeconds      = flag.Float64("begin", 0.0, "beginning of region of interest (seconds)")
	endSeconds        = flag.Float64("end", 0.0, "end of region of interest (seconds)")
	lowFrequency      = flag.Float64("low_freq", 100.0, "lowest frequency of interest")
	highFrequency     = flag.Float64("high_freq", 20000.0, "highest frequency of interest")
	windowSizeSeconds = flag.Float64("window_size_seconds", 0.08, "analysis window size (seconds)")
	analysesPerSecond = flag.Float64("analyses_per_second", 50.0, "number of analysis frames per second")
	threshold         = flag.Float64("threshold", 0.001, "threshold for inclusion (relative to largest coefficient)")
)

func mainCore() error {
	if *inputFile == "" {
		return errors.New("--input is required")
	}

	if *endSeconds <= *beginSeconds {
		return fmt.Errorf("need --begin < -- end: got --begin=%v --end=%v", *beginSeconds, *endSeconds)
	}

	duration := *endSeconds - *beginSeconds
	minDuration := 0.2
	if duration < minDuration {
		return fmt.Errorf("too short (duration must be at least %v, got %v)", minDuration, duration)
	}

	log.Printf("reading input file %q", *inputFile)
	snip, err := snippet.Read(*inputFile)
	if err != nil {
		return err
	}

	snip = snippet.SubsnippetByTime(snip, *beginSeconds, duration)

	anal, err := analysis.Analyze(snip, &analysis.Params{
		MinWindowSizeSeconds:     *windowSizeSeconds,
		NumberOfFrequencyBuckets: 1000,
		Range: &analysis.FrequencyRange{
			LowHz:  *lowFrequency,
			HighHz: *highFrequency,
		},
		PerformPureFFT:    true,
		AnalysesPerSecond: *analysesPerSecond,
	})
	if err != nil {
		return err
	}

	var medians, sortedMedians []float64

	freqs := anal.Points[0].PureFFT.Freqs

	for i := range freqs {
		vc := stats.New()
		for _, point := range anal.Points {
			vc.Add(point.PureFFT.Amplitude[i])
		}
		median := vc.Median()
		medians = append(medians, median)
	}

	for _, x := range medians {
		sortedMedians = append(sortedMedians, x)
	}
	sort.Float64s(sortedMedians)
	maxMedian := sortedMedians[len(sortedMedians)-1]
	log.Printf("largest median is %v", maxMedian)
	correction := 1.0 / maxMedian

	rv := &aborapb.Spectrum{}

	mp := len(anal.Points) / 2

	for i, freq := range freqs {
		if i == 0 {
			continue
		}

		value := medians[i] * correction
		if *threshold > 0 && value < *threshold {
			continue
		}

		phase := anal.Points[mp].PureFFT.Phase[i]

		point := &aborapb.SpectrumPoint{
			Amplitude: value,
			Frequency: freq,
			Phase:     phase,
		}
		rv.Points = append(rv.Points, point)
	}

	rvText := proto.MarshalTextString(rv)
	fmt.Println(rvText)

	return nil
}

func main() {
	flag.Parse()

	if err := mainCore(); err != nil {
		log.Fatalf("failure: %v", err)
	}
}
