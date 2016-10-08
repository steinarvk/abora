package colorscale

import (
	"image/color"
)

type rgb struct {
	red, green, blue float64
}

var simpleRGB = []rgb{
	{0.0, 0.0, 0.0},
	{1.0, 0.0, 0.0},
	{1.0, 1.0, 0.0},
	{1.0, 1.0, 1.0},
}

func interpolateFloatToUnsigned(t, x0, x1 float64) uint16 {
	v := (x0 + t*(x1-x0)) * 0xffff
	if v <= 0 {
		return 0
	}
	if v >= 0xffff {
		return 0xffff
	}
	return uint16(v)
}

func interpolate(xs []rgb, t float64) color.Color {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	tb := float64(len(xs)) * t
	ti := int(tb)
	if ti >= len(xs) {
		ti--
	}
	tt := tb - float64(ti)
	tin := ti + 1
	if tin >= len(xs) {
		tin = ti
	}
	return color.NRGBA64{
		R: interpolateFloatToUnsigned(tt, xs[ti].red, xs[tin].red),
		G: interpolateFloatToUnsigned(tt, xs[ti].green, xs[tin].green),
		B: interpolateFloatToUnsigned(tt, xs[ti].blue, xs[tin].blue),
		A: 0xffff,
	}
}

func Simple(t float64) color.Color {
	return interpolate(simpleRGB, t)
}
