package varying

type Varying interface {
	Value() float64
	Advance(float64)
}

type Constant float64

func (c Constant) Value() float64    { return float64(c) }
func (_ Constant) Advance(_ float64) {}

func Advance(dt float64, xs ...Varying) {
	for _, x := range xs {
		if x != nil {
			x.Advance(dt)
		}
	}
}
