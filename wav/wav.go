package wav

import (
	"fmt"
	"log"
	"math"
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

	worst := float64(0.0)

	var frames int64

	for x := range ch {
		frames++
		xa := math.Abs(x)
		if xa > worst {
			worst = xa
		}
		val := int32(float64(2147483648) * x)
		if err := wr.WriteInt32(val); err != nil {
			f.Close()
			return err
		}
	}

	if err := wr.Close(); err != nil {
		return err
	}

	log.Printf("wrote WAV file %q (%d frames, %v seconds, largest: %v)", filename, frames, float64(frames)/float64(sampleRate), worst)
	if worst > 1.0 {
		return fmt.Errorf("wrote WAV file %q with clipping (%v > %v)", filename, worst, 1.0)
	}

	return nil
}
