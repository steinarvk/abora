package wav

import (
	"os"

	"github.com/cryptix/wav"
)

func WriteFile(filename string, sampleRate int, ch <-chan float64) error {
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
