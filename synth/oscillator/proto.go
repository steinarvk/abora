package oscillator

import (
	"fmt"
	"log"

	pb "github.com/steinarvk/abora/proto"
)

func FromProto(spec *pb.Oscillator) (Oscillator, error) {
	log.Printf("loading oscillator: %v", spec)
	switch opts := spec.Oscillators.(type) {
	default:
		return nil, fmt.Errorf("unhandled kind of oscillator: %v", spec)
	case *pb.Oscillator_Sine:
		return Sin(), nil
	case *pb.Oscillator_Spectrum:
		return FromSpectrum(opts.Spectrum), nil
	}
}
