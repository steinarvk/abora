package envelope

import (
	"fmt"

	pb "github.com/steinarvk/abora/proto"
)

func FromProto(spec *pb.Envelope, duration float64) (Envelope, error) {
	switch opts := spec.EnvelopeKind.(type) {
	default:
		return nil, fmt.Errorf("unhandled kind of envelope: %v", spec)
	case *pb.Envelope_Adsr:
		return LinearADSR(duration, ADSRSpec{
			AttackDuration:  opts.Adsr.AttackDuration,
			DecayDuration:   opts.Adsr.DecayDuration,
			SustainLevel:    opts.Adsr.SustainLevel,
			ReleaseDuration: opts.Adsr.ReleaseDuration,
		}), nil
	}
}
