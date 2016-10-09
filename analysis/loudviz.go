package analysis

import (
	"image"
	"image/color"
	"math"
)

func (a *LoudnessAnalysis) DefaultValueMapper() func(x float64) float64 {
	nearmax := a.ValueStats.Quantile(0.9)
	threshold := a.ValueStats.Quantile(0.1)

	logThreshold := math.Log(threshold)
	logDenom := math.Log(nearmax) - logThreshold

	f := func(x float64) float64 {
		rv := 0.0
		if x > threshold {
			rv = (math.Log(x) - logThreshold) / logDenom
			rv = (x - threshold) / (nearmax - threshold)
		}
		return rv
	}

	return f
}

func (a *LoudnessAnalysis) Visualize(height int, mapper func(float64) float64, colorizer func(float64) color.Color) image.Image {
	width := len(a.Values)
	img := image.NewRGBA64(image.Rect(0, 0, width, height))

	for x, val := range a.Values {
		col := colorizer(mapper(val))
		for y := 0; y < height; y++ {
			img.Set(x, y, col)
		}
	}

	return img
}
