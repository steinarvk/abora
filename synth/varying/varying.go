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

type mappedVarying struct {
	f func(float64) float64
	u Varying
}

func (m *mappedVarying) Value() float64 {
	return m.f(m.u.Value())
}

func (m *mappedVarying) Advance(dt float64) {
	m.u.Advance(dt)
}

func Map(u Varying, f func(float64) float64) Varying {
	return &mappedVarying{
		f: f,
		u: u,
	}
}
