package analysis

import (
	"image"
	"image/color"
	"math"
)

func (a *Analysis) DefaultValueMapper() func(x float64) float64 {
	nearmax := a.ValueStats.Quantile(0.995) * 0.99
	threshold := a.ValueStats.Quantile(0.8)

	logThreshold := math.Log(threshold)
	logDenom := math.Log(nearmax) - logThreshold

	f := func(x float64) float64 {
		rv := 0.0
		if x > threshold {
			rv = (math.Log(x) - logThreshold) / logDenom
		}
		return rv
	}

	return f
}

func (a *Analysis) Visualize(mapper func(float64) float64, colorizer func(float64) color.Color) image.Image {
	width := len(a.Points)
	height := len(a.FrequencyBuckets)
	img := image.NewRGBA64(image.Rect(0, 0, width, height))

	for x, point := range a.Points {
		for i := range a.FrequencyBuckets {
			y := height - 1 - i
			img.Set(x, y, colorizer(mapper(point.Values[i])))
		}
	}

	return img
}
