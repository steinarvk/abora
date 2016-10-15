package analysis

import (
	"log"
	"math"
	"testing"

	"github.com/steinarvk/abora/snippet"
)

type testSnippet struct {
	tone       float64
	sampleRate int
	phase      int
	samples    int
}

func (s *testSnippet) SampleRate() int   { return s.sampleRate }
func (s *testSnippet) TotalSamples() int { return s.samples }
func (s *testSnippet) Subsnippet(i, sz int) snippet.Snippet {
	return &testSnippet{
		tone:    s.tone,
		phase:   s.phase + i,
		samples: sz,
	}
}
func (s *testSnippet) Slice(i, sz int) []float64 {
	var rv []float64
	for index := i; index < (i+sz) && index < s.samples; index++ {
		t := float64(s.phase+index) / float64(s.SampleRate())
		rv = append(rv, math.Sin(2*math.Pi*t*s.tone))
	}
	return rv
}

func TestSanityCheck(t *testing.T) {
	tone := 1444.4
	snip := &testSnippet{tone: tone, sampleRate: 44100, samples: 80000}
	xs := snip.Slice(0, 80000)
	if len(xs) != 80000 {
		t.Fatalf("Slice(0, 80000) = %v (len %d) wanted len %d", xs, len(xs), 80000)
	}
	var max, min float64
	for _, x := range xs {
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	if max < 0.99 {
		t.Errorf("max(xs) = %v want ~= 1", max)
	}
	if min > -0.99 {
		t.Errorf("min(xs) = %v want ~= -1", min)
	}
	c := 0
	for _ = range snippet.Scan(snip) {
		c++
	}
	if c != snip.samples {
		t.Errorf("snippet.Scan() gave wrong length: got %v want %v", c, snip.samples)
	}
}

func TestAnalyzeTone(t *testing.T) {
	tone := 1444.4
	snip := &testSnippet{tone: tone, sampleRate: 44100, samples: 80000}

	anal, err := Analyze(snip, nil)
	if err != nil {
		t.Fatalf("unable to analyze test snippet: %v", err)
	}

	whichBucket := -1
	for index, bucket := range anal.FrequencyBuckets {
		if bucket.LowHz < tone && tone < bucket.HighHz {
			if whichBucket != -1 {
				t.Errorf("more than one bucket found matching tone %v: %v and %v", tone, bucket, anal.FrequencyBuckets[whichBucket])
			}
			whichBucket = index
		}
	}
	if whichBucket == -1 {
		t.Fatalf("no bucket found matching tone %v (out of %d buckets)", tone, len(anal.FrequencyBuckets))
	}

	log.Printf("bucket %v contains %v", anal.FrequencyBuckets[whichBucket], tone)
	log.Printf("checking %d points", len(anal.Points))

	if len(anal.Points) < 1 {
		t.Errorf("expected at least one point, got none")
	}

	for i, point := range anal.Points {
		heaviestBucket := 0
		for j := range anal.FrequencyBuckets[1:] {
			if point.Values[j] > point.Values[heaviestBucket] {
				heaviestBucket = j
			}
		}

		dist := math.Abs(tone - anal.FrequencyBuckets[heaviestBucket].Midpoint())

		if dist > 20 {
			t.Errorf("on %d: expected heaviest bucket close to %v but %v (weight was %v)", i, tone, anal.FrequencyBuckets[heaviestBucket], point.Values[heaviestBucket])
		}
	}
}

func TestAnalyzeTone192(t *testing.T) {
	tone := 1444.4
	snip := &testSnippet{tone: tone, sampleRate: 192000, samples: 80000}

	anal, err := Analyze(snip, nil)
	if err != nil {
		t.Fatalf("unable to analyze test snippet: %v", err)
	}

	whichBucket := -1
	for index, bucket := range anal.FrequencyBuckets {
		if bucket.LowHz < tone && tone < bucket.HighHz {
			if whichBucket != -1 {
				t.Errorf("more than one bucket found matching tone %v: %v and %v", tone, bucket, anal.FrequencyBuckets[whichBucket])
			}
			whichBucket = index
		}
	}
	if whichBucket == -1 {
		t.Fatalf("no bucket found matching tone %v (out of %d buckets)", tone, len(anal.FrequencyBuckets))
	}

	log.Printf("bucket %v contains %v", anal.FrequencyBuckets[whichBucket], tone)
	log.Printf("checking %d points", len(anal.Points))

	if len(anal.Points) < 1 {
		t.Errorf("expected at least one point, got none")
	}

	for i, point := range anal.Points {
		heaviestBucket := 0
		for j := range anal.FrequencyBuckets[1:] {
			if point.Values[j] > point.Values[heaviestBucket] {
				heaviestBucket = j
			}
		}

		dist := math.Abs(tone - anal.FrequencyBuckets[heaviestBucket].Midpoint())

		if dist > 20 {
			t.Errorf("on %d: expected heaviest bucket close to %v but %v (weight was %v)", i, tone, anal.FrequencyBuckets[heaviestBucket], point.Values[heaviestBucket])
		}
	}
}
