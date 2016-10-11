// Package snippet implements an in-memory sound snippet.
package snippet

import (
	"errors"
	"os"

	"azul3d.org/engine/audio"
	_ "azul3d.org/engine/audio/flac"
)

type Snippet interface {
	SampleRate() int
	TotalSamples() int
	Slice(i, j int) []float64
	Subsnippet(i, j int) Snippet
}

func Duration(s Snippet) float64 {
	return float64(s.TotalSamples()) / float64(s.SampleRate())
}

func Scan(s Snippet) <-chan float64 {
	sz := 4096
	ch := make(chan float64, sz)
	index := 0
	total := s.TotalSamples()

	go func() {
		for index < total {
			for _, x := range s.Slice(index, sz) {
				ch <- x
			}
			index += sz
		}
		close(ch)
	}()

	return ch
}

func SubsnippetByTime(s Snippet, t float64, secs float64) Snippet {
	fr := int(float64(s.SampleRate()) * t)
	w := int(float64(s.SampleRate()) * secs)
	return s.Subsnippet(fr, w)
}

type inMemorySnippet struct {
	sampleRate int
	samples    []float64
}

func (m *inMemorySnippet) Subsnippet(i, sz int) Snippet {
	return &inMemorySnippet{
		sampleRate: m.sampleRate,
		samples:    m.Slice(i, sz),
	}
}

func (m *inMemorySnippet) SampleRate() int {
	return m.sampleRate
}

func (m *inMemorySnippet) TotalSamples() int {
	return len(m.samples)
}

func (m *inMemorySnippet) Slice(i, sz int) []float64 {
	j := i + sz
	if j >= len(m.samples) {
		j = len(m.samples)
	}
	return m.samples[i:j]
}

func Read(filename string) (Snippet, error) {
	fileHandle, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	decoder, _, err := audio.NewDecoder(fileHandle)
	if err != nil {
		fileHandle.Close()
		return nil, err
	}

	config := decoder.Config()

	if config.Channels != 1 {
		return nil, errors.New("expected mono input")
	}

	seconds := 1
	bufsize := seconds * config.SampleRate
	underlying := audio.Float64{}
	buf := underlying.Make(bufsize, bufsize)

	rv := &inMemorySnippet{
		sampleRate: config.SampleRate,
	}

	for {
		read, err := decoder.Read(buf)
		if err != nil && err != audio.EOS {
			return nil, err
		}

		for i := 0; i < read; i++ {
			val := buf.At(i)
			rv.samples = append(rv.samples, val)
		}

		if err == audio.EOS {
			break
		}
	}

	return rv, nil
}
