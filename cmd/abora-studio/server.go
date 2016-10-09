package main

import (
	"errors"
	"flag"
	"fmt"
	"image/png"
	"log"
	"net/http"

	"github.com/steinarvk/abora/analysis"
	"github.com/steinarvk/abora/colorscale"
	"github.com/steinarvk/abora/snippet"
)

var (
	inputFilename = flag.String("input", "", "input filename")
	port          = flag.Int("port", 8099, "port on which to listen")
)

type studioServer struct {
	snip snippet.Snippet
}

func (s *studioServer) getSnippet(req *http.Request) (snippet.Snippet, error) {
	params := &paramGetter{req, nil}

	t := params.getFloat("t", 0.0)
	dur := params.getFloat("duration", 5.0)

	if params.err != nil {
		return nil, params.err
	}

	return snippet.SubsnippetByTime(s.snip, t, dur), nil
}

func (s *studioServer) serveSpectrogram(w http.ResponseWriter, req *http.Request) error {
	v := req.URL.Query()
	log.Printf("serving spectrogram request: %v", v)
	defer log.Printf("done serving spectrogram request: %v", v)

	snip, err := s.getSnippet(req)
	if err != nil {
		return err
	}

	params := &paramGetter{req, nil}

	timeRes := params.getFloat("timeRes", 100.0)
	freqRes := params.getInt("freqRes", 1000)
	lowHz := params.getFloat("lowHz", 500.0)
	highHz := params.getFloat("highHz", 5000.0)
	windowSize := params.getFloat("windowSize", 0.05)

	if params.err != nil {
		return params.err
	}

	anal, err := analysis.Analyze(snip, &analysis.Params{
		MinWindowSizeSeconds:     windowSize,
		NumberOfFrequencyBuckets: freqRes,
		Range: &analysis.FrequencyRange{
			LowHz:  lowHz,
			HighHz: highHz,
		},
		AnalysesPerSecond: timeRes,
	})
	if err != nil {
		return err
	}

	img := anal.Visualize(anal.DefaultValueMapper(), colorscale.Viridis)

	w.Header().Set("Content-Type", "image/png")

	if err := png.Encode(w, img); err != nil {
		log.Printf("write/encode error: %v", err)
		return err
	}

	return nil
}

func serveErrorOr(f func(http.ResponseWriter, *http.Request) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if err := f(w, req); err != nil {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("error: %v", err)))
		}
	}
}

func mainCore() error {
	if *inputFilename == "" {
		return errors.New("--input is required")
	}

	log.Printf("reading %q", *inputFilename)
	snip, err := snippet.Read(*inputFilename)
	if err != nil {
		return err
	}

	serv := &studioServer{snip}

	http.HandleFunc("/spectrogram", serveErrorOr(serv.serveSpectrogram))

	servePattern := fmt.Sprintf(":%d", *port)
	log.Printf("listening on %q", servePattern)
	log.Printf("link: http://localhost:%d/spectrogram?t=0&duration=5", *port)

	return http.ListenAndServe(servePattern, nil)
}

func main() {
	flag.Parse()

	if err := mainCore(); err != nil {
		log.Fatalf("failure: %v", err)
	}
}
