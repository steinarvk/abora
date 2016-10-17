package oscillator

import (
	"fmt"

	pb "github.com/steinarvk/abora/proto"
)

func FromProto(spec *pb.Oscillator) (Oscillator, error) {
	switch spec.Oscillators.(type) {
	default:
		return nil, fmt.Errorf("unhandled kind of oscillator: %v", spec)
	case *pb.Oscillator_Sine:
		return Sin(), nil
	}
}
