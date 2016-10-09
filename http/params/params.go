package params

import (
	"log"
	"net/http"
	"strconv"
)

type ParamGetter struct {
	r   *http.Request
	err error
}

func Getter(r *http.Request) *ParamGetter {
	return &ParamGetter{r, nil}
}

func (p *ParamGetter) Err() error {
	return p.err
}

func (p *ParamGetter) Float(name string, defValue float64) float64 {
	v, err := floatParam(p.r, name, defValue)
	if err != nil {
		p.err = err
	}
	return v
}

func (p *ParamGetter) Int(name string, defValue int) int {
	v, err := intParam(p.r, name, defValue)
	if err != nil {
		p.err = err
	}
	return v
}

func intParam(req *http.Request, name string, defaultValue int) (int, error) {
	val := req.URL.Query().Get(name)
	if val == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("failed to parse %q=%q: %v", name, val, err)
		return defaultValue, err
	}

	return parsed, nil
}

func floatParam(req *http.Request, name string, defaultValue float64) (float64, error) {
	val := req.URL.Query().Get(name)
	if val == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		log.Printf("failed to parse %q=%q: %v", name, val, err)
		return defaultValue, err
	}

	return parsed, nil
}
