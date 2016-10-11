package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"path/filepath"

	"github.com/steinarvk/abora/analysis"
	"github.com/steinarvk/abora/colorscale"
	"github.com/steinarvk/abora/http/params"
	"github.com/steinarvk/abora/snippet"
)

var (
	inputFilename = flag.String("input", "", "input filename")
	port          = flag.Int("port", 8099, "port on which to listen")
	staticFiles   = flag.String("static_files_dir", "./static/", "directory with static files")
)

type studioServer struct {
	snip snippet.Snippet
}

func (s *studioServer) getSnippet(req *http.Request) (snippet.Snippet, error) {
	params := params.Getter(req)

	t := params.Float("t", 0.0)
	dur := params.Float("duration", 5.0)

	if params.Err() != nil {
		return nil, params.Err()
	}

	return snippet.SubsnippetByTime(s.snip, t, dur), nil
}

func (s *studioServer) getAnalysisParams(req *http.Request, snip snippet.Snippet) (*analysis.Params, error) {
	params := params.Getter(req)

	timeRes := params.Float("timeRes", 100.0)
	freqRes := params.Int("freqRes", 1000)
	lowHz := params.Float("lowHz", 500.0)
	highHz := params.Float("highHz", 5000.0)
	windowSize := params.Float("windowSize", 0.05)

	if width := params.Int("pxWidth", 0); width > 0 {
		timeRes = float64(width) / snippet.Duration(snip)
	}
	if height := params.Int("pxHeight", 0); height > 0 {
		freqRes = height
	}

	loudnessWindowSize := params.Float("loudnessWindowSize", 0.001)

	if params.Err() != nil {
		return nil, params.Err()
	}

	return &analysis.Params{
		LoudnessWindowSizeSeconds: loudnessWindowSize,
		MinWindowSizeSeconds:      windowSize,
		NumberOfFrequencyBuckets:  freqRes,
		Range: &analysis.FrequencyRange{
			LowHz:  lowHz,
			HighHz: highHz,
		},
		AnalysesPerSecond: timeRes,
	}, nil
}

func (s *studioServer) serveLoudness(w http.ResponseWriter, req *http.Request) error {
	v := req.URL.Query()
	log.Printf("serving loudness request: %v", v)
	defer log.Printf("done serving loudness request: %v", v)

	params := params.Getter(req)

	loudnessHeight := params.Int("loudnessHeight", 100)

	if params.Err() != nil {
		return params.Err()
	}

	snip, err := s.getSnippet(req)
	if err != nil {
		return err
	}

	analParams, err := s.getAnalysisParams(req, snip)
	if err != nil {
		return err
	}

	anal, err := analysis.AnalyzeLoudness(snip, analParams)
	if err != nil {
		return err
	}

	img := anal.Visualize(loudnessHeight, anal.DefaultValueMapper(), colorscale.Viridis)

	w.Header().Set("Content-Type", "image/png")

	if err := png.Encode(w, img); err != nil {
		log.Printf("write/encode error: %v", err)
		return err
	}

	return nil
}

func (s *studioServer) serveSpectrogram(w http.ResponseWriter, req *http.Request) error {
	v := req.URL.Query()
	log.Printf("serving spectrogram request: %v", v)
	defer log.Printf("done serving spectrogram request: %v", v)

	snip, err := s.getSnippet(req)
	if err != nil {
		return err
	}

	params, err := s.getAnalysisParams(req, snip)
	if err != nil {
		return err
	}

	anal, err := analysis.Analyze(snip, params)
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

func (s *studioServer) serveSpectrogramMetadata(w http.ResponseWriter, req *http.Request) error {
	v := req.URL.Query()
	log.Printf("serving spectrogram metadata request: %v", v)
	defer log.Printf("done serving spectrogram metadata request: %v", v)

	snip, err := s.getSnippet(req)
	if err != nil {
		return err
	}

	params, err := s.getAnalysisParams(req, snip)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	rv := struct {
		TimeResolution   float64
		LowFrequency     float64
		HighFrequency    float64
		FrequencyBuckets int
	}{
		TimeResolution:   params.AnalysesPerSecond,
		FrequencyBuckets: params.NumberOfFrequencyBuckets,
		LowFrequency:     params.Range.LowHz,
		HighFrequency:    params.Range.HighHz,
	}

	return encoder.Encode(&rv)
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

	http.HandleFunc("/spectrogram/png", serveErrorOr(serv.serveSpectrogram))
	http.HandleFunc("/spectrogram/metadata", serveErrorOr(serv.serveSpectrogramMetadata))
	http.HandleFunc("/loudness", serveErrorOr(serv.serveLoudness))

	staticFiles, err := filepath.Abs(*staticFiles)
	if err != nil {
		return err
	}

	serveStatic := func(fn string) func(http.ResponseWriter, *http.Request) {
		path := filepath.Join(staticFiles, fn)
		return func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, path)
		}
	}

	http.HandleFunc("/js/editor.js", serveStatic("/js/editor.js"))
	http.HandleFunc("/", serveStatic("/index.html"))

	servePattern := fmt.Sprintf(":%d", *port)
	log.Printf("listening on %q", servePattern)
	log.Printf("link: http://localhost:%d/spectrogram/png?t=0&duration=5", *port)

	return http.ListenAndServe(servePattern, nil)
}

func main() {
	flag.Parse()

	if err := mainCore(); err != nil {
		log.Fatalf("failure: %v", err)
	}
}
