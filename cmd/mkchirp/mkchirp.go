package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"

	"github.com/golang/protobuf/proto"

	"github.com/steinarvk/abora/synth/chirp"
	"github.com/steinarvk/abora/synth/mix"
	"github.com/steinarvk/abora/wav"

	aborapb "github.com/steinarvk/abora/proto"
)

var (
	outputFilename = flag.String("output", "", "output filename")
	fromProtoFile  = flag.String("proto", "", "proto filename")
)

func mainCore() error {
	if *outputFilename == "" {
		return errors.New("--output is required")
	}

	if *fromProtoFile == "" {
		return errors.New("--proto is required")
	}

	var chrp *chirp.TimedChirp

	if *fromProtoFile != "" {
		data, err := ioutil.ReadFile(*fromProtoFile)
		if err != nil {
			return err
		}

		spec := &aborapb.Chirp{}
		if err := proto.UnmarshalText(string(data), spec); err != nil {
			return err
		}

		chrp, err = chirp.FromProto(spec, nil)
		if err != nil {
			return err
		}
	}

	sampleRate := 44110

	ch := mix.AsChannel(
		[]chirp.TimedChirp{*chrp},
		sampleRate,
		0.0)

	return wav.WriteFile(*outputFilename, sampleRate, ch)
}

func main() {
	flag.Parse()

	if err := mainCore(); err != nil {
		log.Fatalf("failure: %v", err)
	}
}
