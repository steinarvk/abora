package analysis

import (
	"log"
	"math"
	"math/cmplx"

	"github.com/steinarvk/abora/snippet"
	"github.com/steinarvk/abora/stats"

	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/spectral"
)

type FrequencyRange struct {
	LowHz  float64
	HighHz float64
}

func (x FrequencyRange) Midpoint() float64 {
	return 0.5 * (x.LowHz + x.HighHz)
}

type Params struct {
	MinWindowSizeSeconds      float64
	LoudnessWindowSizeSeconds float64
	AnalysesPerSecond         float64
	Range                     *FrequencyRange
	NumberOfFrequencyBuckets  int
	PwelchPadding             *int
	PerformPureFFT            bool
}

var (
	defaultPwelchPadding = 8192

	defaultParams = Params{
		MinWindowSizeSeconds:      0.05,
		LoudnessWindowSizeSeconds: 0.005,
		AnalysesPerSecond:         100.0,
		Range: &FrequencyRange{
			LowHz:  500.0,
			HighHz: 5000.0,
		},
		PwelchPadding:            &defaultPwelchPadding,
		NumberOfFrequencyBuckets: 1000,
	}
)

type PureFFTPoint struct {
	Raw       []complex128
	Freqs     []float64
	Amplitude []float64
	Phase     []float64
}

type AnalysisPoint struct {
	FrameNumber int
	Values      []float64
	RawPXX      []float64
	RawFreqs    []float64
	PureFFT     *PureFFTPoint
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

type LoudnessAnalysis struct {
	Params                *Params
	SampleRate            int
	Values                []float64
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

	if params.LoudnessWindowSizeSeconds == 0 {
		params.LoudnessWindowSizeSeconds = defaultParams.LoudnessWindowSizeSeconds
	}

	if params.AnalysesPerSecond == 0 {
		params.AnalysesPerSecond = defaultParams.AnalysesPerSecond
	}

	if params.PwelchPadding == nil {
		params.PwelchPadding = defaultParams.PwelchPadding
	}

	if params.NumberOfFrequencyBuckets == 0 {
		params.NumberOfFrequencyBuckets = defaultParams.NumberOfFrequencyBuckets
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

	if rv.Pad != 0 && rv.Pad < rv.NFFT {
		log.Printf("error: PwelchOptions: Pad (%v) must not be smaller than NFFT (%v)", rv.Pad, rv.NFFT)
		rv.Pad = rv.NFFT
		log.Printf("error: PwelchOptions: correction applied: Pad (%v) NFFT (%v)", rv.Pad, rv.NFFT)
	}
	return rv
}

func (a *Analysis) newPoint(sampleNo int, pxx, freqs []float64) (*AnalysisPoint, error) {
	point := &AnalysisPoint{
		FrameNumber: sampleNo,
		RawPXX:      pxx,
		RawFreqs:    freqs,
		Values:      make([]float64, len(a.FrequencyBuckets)),
	}
	for i, bucket := range a.FrequencyBuckets {
		value := bucket.density(point.RawPXX, point.RawFreqs)
		point.Values[i] = value
		a.ValueStats.Add(value)
	}
	return point, nil
}

func rootMeanSquare(xs []float64) float64 {
	var rv float64
	for _, x := range xs {
		rv += x * x
	}
	return math.Sqrt(rv / float64(len(xs)))
}

func (a *LoudnessAnalysis) addPoint(_ int64, frames []float64) error {
	value := rootMeanSquare(frames)
	a.Values = append(a.Values, value)
	a.ValueStats.Add(value)
	return nil
}

func calculatePureFFT(frames []float64, sampleRate int) *PureFFTPoint {
	val := fft.FFTReal(frames)
	N := len(frames)
	rv := &PureFFTPoint{}
	// Note FFTReal output is a "mirror image"; the last half contains no new information.
	for k, x := range val[:len(val)/2] {
		// k cycles per sampleRate samples
		// e.g. if k=10 then 10 cycles per 44100 samples
		// which is 10 cycles per second
		// which is 10 Hz
		freq := float64(k) * float64(sampleRate) / float64(N)
		amp := cmplx.Abs(x) / float64(N)
		phase := cmplx.Phase(x)
		rv.Raw = append(rv.Raw, x)
		rv.Freqs = append(rv.Freqs, freq)
		rv.Amplitude = append(rv.Amplitude, amp)
		rv.Phase = append(rv.Phase, phase)
	}
	return rv
}

func (a *Analysis) addPoint(sampleNo int64, frames []float64) error {
	//	log.Printf("performing spectral.Pwelch([...%d...], %v, %v)", len(frames), a.SampleRate, a.pwelchOpts())
	pxx, freqs := spectral.Pwelch(frames, float64(a.SampleRate), a.pwelchOpts())
	point, err := a.newPoint(int(sampleNo), pxx, freqs)
	if err != nil {
		return err
	}

	if a.Params.PerformPureFFT {
		point.PureFFT = calculatePureFFT(frames, a.SampleRate)
	}

	a.Points = append(a.Points, point)

	return nil
}

func AnalyzeLoudness(s snippet.Snippet, params *Params) (*LoudnessAnalysis, error) {
	if params == nil {
		params = &Params{}
	}
	if err := normalizeParams(s, params); err != nil {
		return nil, err
	}

	rv := &LoudnessAnalysis{
		Params:     params,
		SampleRate: s.SampleRate(),
		ValueStats: stats.New(),
	}

	rv.WindowSize = int(float64(s.SampleRate()) * params.LoudnessWindowSizeSeconds)
	rv.FramesBetweenAnalyses = int(float64(s.SampleRate()) / params.AnalysesPerSecond)

	if err := onWindows(rv.WindowSize, rv.FramesBetweenAnalyses, snippet.Scan(s), rv.addPoint); err != nil {
		return nil, err
	}

	return rv, nil
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
