.PHONY: all clean dependencies reader writer

all: reader writer

reader:
	go build github.com/steinarvk/abora/cmd/reader

writer:
	go build github.com/steinarvk/abora/cmd/writer

clean:
	rm -f reader writer

dependencies:
	go get azul3d.org/engine/audio
	go get azul3d.org/engine/audio/flac
	go get github.com/mjibson/go-dsp/spectral
	go get github.com/cryptix/wav
