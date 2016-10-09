package analysis

import (
	"github.com/steinarvk/abora/snippet"
	"github.com/steinarvk/abora/stats"
	"log"

	"github.com/mjibson/go-dsp/spectral"
)

type FrequencyRange struct {
	LowHz  float64
	HighHz float64
}

type Params struct {
	MinWindowSizeSeconds     float64
	AnalysesPerSecond        float64
	Range                    *FrequencyRange
	NumberOfFrequencyBuckets int
	PwelchPadding            *int
}

var (
	defaultPwelchPadding = 8192

	defaultParams = Params{
		MinWindowSizeSeconds: 0.05,
		AnalysesPerSecond:    100.0,
		Range: &FrequencyRange{
			LowHz:  500.0,
			HighHz: 5000.0,
		},
		PwelchPadding:            &defaultPwelchPadding,
		NumberOfFrequencyBuckets: 1000,
	}
)

type AnalysisPoint struct {
	FrameNumber int
	Values      []float64
	rawPXX      []float64
	rawFreqs    []float64
}

type Analysis struct {
	Params                *Params
	FrequencyBuckets      []FrequencyRange
	SampleRate            int
	Points                []*AnalysisPoint
	ValueStats            *stats.ValueCollection
	WindowSize            int
	FramesBetweenAnalyses int
}

func (r FrequencyRange) amountInside(f0, f1 float64) float64 {
	if r.LowHz >= f1 {
		return 0.0
	}
	if r.HighHz <= f0 {
		return 0.0
	}
	if r.LowHz > f0 {
		f0 = r.LowHz
	}
	if r.HighHz < f1 {
		f1 = r.HighHz
	}
	return f1 - f0
}

func (r FrequencyRange) density(pxx, freqs []float64) float64 {
	var total float64
	for i, w := range pxx[:len(pxx)-1] {
		f0 := freqs[i]
		f1 := freqs[i+1]
		total += r.amountInside(f0, f1) * w
	}
	return total
}

func (r FrequencyRange) Subdivide(n int) []FrequencyRange {
	span := r.HighHz - r.LowHz

	var rv []FrequencyRange
	for i := 0; i < n; i++ {
		low := r.LowHz + span*float64(i)/float64(n)
		high := r.LowHz + span*float64(i+1)/float64(n)
		rv = append(rv, FrequencyRange{
			LowHz:  low,
			HighHz: high,
		})
	}

	return rv
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

func nextPowerOfTwo(t float64) int {
	rv := 1
	for float64(rv) < t {
		rv *= 2
	}
	return rv
}

func normalizeParams(s snippet.Snippet, params *Params) error {
	if params.Range == nil {
		params.Range = defaultParams.Range
	}

	if params.MinWindowSizeSeconds == 0 {
		params.MinWindowSizeSeconds = defaultParams.MinWindowSizeSeconds
	}

	if params.AnalysesPerSecond == 0 {
		params.AnalysesPerSecond = defaultParams.AnalysesPerSecond
	}

	if params.PwelchPadding == nil {
		params.PwelchPadding = defaultParams.PwelchPadding
	}

	return nil
}

func (a *Analysis) pwelchOpts() *spectral.PwelchOptions {
	rv := &spectral.PwelchOptions{
		NFFT: a.WindowSize,
	}
	if a.Params.PwelchPadding != nil {
		rv.Pad = *a.Params.PwelchPadding
	}
	return rv
}

func (a *Analysis) newPoint(sampleNo int, pxx, freqs []float64) (*AnalysisPoint, error) {
	point := &AnalysisPoint{
		FrameNumber: sampleNo,
		rawPXX:      pxx,
		rawFreqs:    freqs,
		Values:      make([]float64, len(a.FrequencyBuckets)),
	}
	for i, bucket := range a.FrequencyBuckets {
		value := bucket.density(point.rawPXX, point.rawFreqs)
		point.Values[i] = value
		a.ValueStats.Add(value)
	}
	return point, nil
}

func (a *Analysis) addPoint(sampleNo int64, frames []float64) error {
	pxx, freqs := spectral.Pwelch(frames, float64(a.SampleRate), a.pwelchOpts())
	point, err := a.newPoint(int(sampleNo), pxx, freqs)
	if err != nil {
		return err
	}
	a.Points = append(a.Points, point)
	return nil
}

func Analyze(s snippet.Snippet, params *Params) (*Analysis, error) {
	if params == nil {
		params = &Params{}
	}
	if err := normalizeParams(s, params); err != nil {
		return nil, err
	}

	rv := &Analysis{
		Params:           params,
		FrequencyBuckets: params.Range.Subdivide(params.NumberOfFrequencyBuckets),
		SampleRate:       s.SampleRate(),
		ValueStats:       stats.New(),
	}

	rv.WindowSize = nextPowerOfTwo(float64(s.SampleRate()) * params.MinWindowSizeSeconds)
	rv.FramesBetweenAnalyses = int(float64(s.SampleRate()) / params.AnalysesPerSecond)

	log.Printf("window size %d everyNth %d opts %v", rv.WindowSize, rv.FramesBetweenAnalyses, rv.pwelchOpts())

	if err := onWindows(rv.WindowSize, rv.FramesBetweenAnalyses, snippet.Scan(s), rv.addPoint); err != nil {
		return nil, err
	}

	return rv, nil
}
