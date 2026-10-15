package decisions

// Resolved is a reviewed forecast: the chance given that the expected
// outcome would happen, and whether it did.
type Resolved struct {
	P        float64
	Happened bool
}

// Bucket is the forecasts whose chance fell in [From, From+0.1).
type Bucket struct {
	From      float64 `json:"from"`
	Count     int     `json:"count"`
	Predicted float64 `json:"predicted"` // mean chance given
	Happened  float64 `json:"happened"`  // share that came true
}

// Calibration is how well a person's forecasts matched what happened.
type Calibration struct {
	Count     int      `json:"count"`
	Brier     *float64 `json:"brier"` // mean squared error: 0 is perfect, always saying 50% scores 0.25
	Predicted float64  `json:"predicted"`
	Happened  float64  `json:"happened"`
	Buckets   []Bucket `json:"buckets"` // the non-empty tenths, lowest first
}

// Score computes the Brier score and the reliability buckets.
func Score(rs []Resolved) Calibration {
	c := Calibration{Buckets: []Bucket{}}
	var bs [10]Bucket
	var brier float64
	for _, r := range rs {
		o := 0.0
		if r.Happened {
			o = 1
		}
		i := min(int(r.P*10+1e-9), 9)
		bs[i].Count++
		bs[i].Predicted += r.P
		bs[i].Happened += o
		c.Predicted += r.P
		c.Happened += o
		brier += (r.P - o) * (r.P - o)
	}
	c.Count = len(rs)
	if c.Count == 0 {
		return c
	}
	n := float64(c.Count)
	brier /= n
	c.Brier, c.Predicted, c.Happened = &brier, c.Predicted/n, c.Happened/n
	for i, b := range bs {
		if b.Count > 0 {
			b.From = float64(i) / 10
			b.Predicted /= float64(b.Count)
			b.Happened /= float64(b.Count)
			c.Buckets = append(c.Buckets, b)
		}
	}
	return c
}
