package mix

import (
	"github.com/steinarvk/abora/synth/chirp"

	"github.com/bradfitz/slice"
)

func AsChannel(chirps []chirp.TimedChirp, sampleRate int, timeLimit float64) <-chan float64 {
	pending := chirps
	slice.Sort(pending, func(i, j int) bool {
		return chirps[i].Time < chirps[j].Time
	})

	bufsz := sampleRate
	ch := make(chan float64, bufsz)

	go func() {
		var active []chirp.Chirp
		frame := 0
		step := 1.0 / float64(sampleRate)

		for len(active) > 0 || len(pending) > 0 {
			t := float64(frame) / float64(sampleRate)
			if timeLimit > 0 && t > timeLimit {
				break
			}

			for len(pending) > 0 && pending[0].Time <= t {
				active = append(active, pending[0].Chirp)
				pending = pending[1:]
			}

			rv := 0.0

			var newActive []chirp.Chirp
			for _, chirp := range active {
				chirp.Advance(step)
				rv += chirp.Sample()
				if !chirp.Done() {
					newActive = append(newActive, chirp)
				}
			}
			active = newActive

			ch <- rv

			frame++
		}

		close(ch)
	}()

	return ch
}
