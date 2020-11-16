package generic

import (
	"fmt"
	"math"

	"go.uber.org/atomic"
)

type bucket struct {
	atomic.Float64
	upper float64 // bucket upper bound, inclusive
}

type Buckets []*bucket

func NewBuckets(upperBounds []float64) Buckets {
	bs := make(Buckets, 0, len(upperBounds)+1)
	for _, upper := range upperBounds {
		bs = append(bs, &bucket{upper: upper})
	}
	if upperBounds[len(upperBounds)-1] != math.MaxFloat64 {
		bs = append(bs, &bucket{upper: math.MaxFloat64})
	}
	return bs
}

func (bs Buckets) get(val float64) *bucket {
	// Binary search to find the correct bucket for this observation. Bucket
	// upper bounds are inclusive.
	i, j := 0, len(bs)
	for i < j {
		h := i + (j-i)/2
		if val > bs[h].upper {
			i = h + 1
		} else {
			j = h
		}
	}
	return bs[i]
}

type Histogram struct {
	Name    string
	Bounds  []float64
	Buckets Buckets
}

func NewHistogram(name string, uppers []float64) *Histogram {
	return &Histogram{
		Buckets: NewBuckets(uppers),
		Bounds:  uppers,
	}
}

func (h *Histogram) Observe(value float64) {
	if h == nil {
		return
	}
	h.IncBucket(value)
}

func (h *Histogram) Value() map[string]interface{} {
	values := make(map[string]interface{})
	for idx, b := range h.Buckets {
		if idx == 0 {
			continue
		}
		tag := fmt.Sprintf("%f-%f", h.Buckets[idx-1].upper, b.upper)
		values[tag] = b.Load()
	}
	return values
}

func (h *Histogram) IncBucket(n float64) {
	if h == nil {
		return
	}
	bucket := h.Buckets.get(n)
	bucket.Add(1.0)
}
