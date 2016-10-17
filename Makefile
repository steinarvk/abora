.PHONY: all clean dependencies reader writer abora-studio mkchirp vowelscan protos

all: protos reader writer abora-studio mkchirp vowelscan

reader:
	go build github.com/steinarvk/abora/cmd/reader

writer:
	go build github.com/steinarvk/abora/cmd/writer

vowelscan:
	go build github.com/steinarvk/abora/cmd/vowelscan

abora-studio:
	go build github.com/steinarvk/abora/cmd/abora-studio

mkchirp:
	go build github.com/steinarvk/abora/cmd/mkchirp

protos:
	protoc proto/abora.proto --go_out=.

clean:
	rm -f reader writer

dependencies:
	go get azul3d.org/engine/audio
	go get azul3d.org/engine/audio/flac
	go get github.com/mjibson/go-dsp/spectral
	go get github.com/cryptix/wav
