// Copyright 2012 Daniel Connelly.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtreego

import (
	"fmt"
	"math"
	"strings"
)

// DistError is an improper distance measurement.  It implements the error
// and is generated when a distance-related assertion fails.
type DistError float64

func (err DistError) Error() string {
	return "rtreego: improper distance"
}

// Point represents a point in 3-dimensional Euclidean space.
type Point [Dim]float64

// Dist computes the Euclidean distance between two points p and q.
func (p Point) dist(q Point) float64 {
	sum := 0.0
	for i := range p {
		dx := p[i] - q[i]
		sum += dx * dx
	}
	return math.Sqrt(sum)
}

// minDist computes the square of the distance from a point to a rectangle.
// If the point is contained in the rectangle then the distance is zero.
//
// Implemented per Definition 2 of "Nearest Neighbor Queries" by
// N. Roussopoulos, S. Kelley and F. Vincent, ACM SIGMOD, pages 71-79, 1995.
func (p Point) minDist(r *Rect) float64 {
	sum := 0.0
	for i, pi := range p {
		if pi < r.p[i] {
			d := pi - r.p[i]
			sum += d * d
		} else if pi > r.q[i] {
			d := pi - r.q[i]
			sum += d * d
		} else {
			sum += 0
		}
	}
	return sum
}

// minMaxDist computes the minimum of the maximum distances from p to points
// on r.  If r is the bounding box of some geometric objects, then there is
// at least one object contained in r within minMaxDist(p, r) of p.
//
// Implemented per Definition 4 of "Nearest Neighbor Queries" by
// N. Roussopoulos, S. Kelley and F. Vincent, ACM SIGMOD, pages 71-79, 1995.
func (p Point) minMaxDist(r *Rect) float64 {
	// by definition, MinMaxDist(p, r) =
	// min{1<=k<=n}(|pk - rmk|^2 + sum{1<=i<=n, i != k}(|pi - rMi|^2))
	// where rmk and rMk are defined as follows:

	rm := func(k int) float64 {
		if p[k] <= (r.p[k]+r.q[k])/2 {
			return r.p[k]
		}
		return r.q[k]
	}

	rM := func(k int) float64 {
		if p[k] >= (r.p[k]+r.q[k])/2 {
			return r.p[k]
		}
		return r.q[k]
	}

	// This formula can be computed in linear time by precomputing
	// S = sum{1<=i<=n}(|pi - rMi|^2).

	S := 0.0
	for i := range p {
		d := p[i] - rM(i)
		S += d * d
	}

	// Compute MinMaxDist using the precomputed S.
	min := math.MaxFloat64
	for k := range p {
		d1 := p[k] - rM(k)
		d2 := p[k] - rm(k)
		d := S - d1*d1 + d2*d2
		if d < min {
			min = d
		}
	}

	return min
}

// Rect represents a subset of 3-dimensional Euclidean space of the form
// [a1, b1] x [a2, b2] x ... x [an, bn], where ai < bi for all 1 <= i <= n.
type Rect struct {
	p, q Point // Enforced by NewRect: p[i] <= q[i] for all i.
}

// The coordinate of the point of the rectangle at i
func (r *Rect) PointCoord(i int) float64 {
	return r.p[i]
}

// The coordinate of the lengths of the rectangle at i
func (r *Rect) LengthsCoord(i int) float64 {
	return r.q[i] - r.p[i]
}

// Equal returns true if the two rectangles are equal
func (r *Rect) Equal(other *Rect) bool {
	for i, e := range r.p {
		if e != other.p[i] {
			return false
		}
	}
	for i, e := range r.q {
		if e != other.q[i] {
			return false
		}
	}
	return true
}

func (r *Rect) String() string {
	var s [Dim]string
	for i, a := range r.p {
		b := r.q[i]
		s[i] = fmt.Sprintf("[%.2f, %.2f]", a, b)
	}
	return strings.Join(s[:], "x")
}

// NewRect constructs and returns a pointer to a Rect given a corner point and
// the lengths of each dimension.  The point p should be the most-negative point
// on the rectangle (in every dimension) and every length should be positive.
func NewRect(p Point, lengths [Dim]float64) (r Rect, err error) {
	r.p = p
	r.q = lengths
	for i, l := range r.q {
		if l <= 0 {
			return r, DistError(l)
		}
		r.q[i] += r.p[i]
	}
	return r, nil
}

// size computes the measure of a rectangle (the product of its side lengths).
func (r *Rect) size() float64 {
	size := 1.0
	for i, a := range r.p {
		b := r.q[i]
		size *= b - a
	}
	return size
}

// margin computes the sum of the edge lengths of a rectangle.
func (r *Rect) margin() float64 {
	// The number of edges in an n-dimensional rectangle is n * 2^(n-1)
	// (http://en.wikipedia.org/wiki/Hypercube_graph).  Thus the number
	// of edges of length (ai - bi), where the rectangle is determined
	// by p = (a1, a2, ..., an) and q = (b1, b2, ..., bn), is 2^(n-1).
	//
	// The margin of the rectangle, then, is given by the formula
	// 2^(n-1) * [(b1 - a1) + (b2 - a2) + ... + (bn - an)].
	sum := 0.0
	for i, a := range r.p {
		b := r.q[i]
		sum += b - a
	}
	return 4.0 * sum
}

// containsPoint tests whether p is located inside or on the boundary of r.
func (r *Rect) containsPoint(p Point) bool {
	for i, a := range p {
		// p is contained in (or on) r if and only if p <= a <= q for
		// every dimension.
		if a < r.p[i] || a > r.q[i] {
			return false
		}
	}

	return true
}

// containsRect tests whether r2 is is located inside r1.
func (r1 *Rect) containsRect(r2 *Rect) bool {
	for i, a1 := range r1.p {
		b1, a2, b2 := r1.q[i], r2.p[i], r2.q[i]
		// enforced by constructor: a1 <= b1 and a2 <= b2.
		// so containment holds if and only if a1 <= a2 <= b2 <= b1
		// for every dimension.
		if a1 > a2 || b2 > b1 {
			return false
		}
	}

	return true
}

func (r1 *Rect) enlarge(r2 *Rect) {
	for i := 0; i < Dim; i++ {
		if r1.p[i] > r2.p[i] {
			r1.p[i] = r2.p[i]
		}
		if r1.q[i] < r2.q[i] {
			r1.q[i] = r2.q[i]
		}
	}
}

// intersect computes the intersection of two rectangles.  If no intersection
// exists, the intersection is nil.
func intersect(r1, r2 *Rect) bool {
	// There are four cases of overlap:
	//
	//     1.  a1------------b1
	//              a2------------b2
	//              p--------q
	//
	//     2.       a1------------b1
	//         a2------------b2
	//              p--------q
	//
	//     3.  a1-----------------b1
	//              a2-------b2
	//              p--------q
	//
	//     4.       a1-------b1
	//         a2-----------------b2
	//              p--------q
	//
	// Thus there are only two cases of non-overlap:
	//
	//     1. a1------b1
	//                    a2------b2
	//
	//     2.             a1------b1
	//        a2------b2
	//
	// Enforced by constructor: a1 <= b1 and a2 <= b2.  So we can just
	// check the endpoints.

	for i := 0; i < Dim; i++ {
		if r2.q[i] <= r1.p[i] || r1.q[i] <= r2.p[i] {
			return false
		}
	}
	return true
}

// ToRect constructs a rectangle containing p with side lengths 2*tol.
func (p Point) ToRect(tol float64) *Rect {
	var r Rect
	for i := range p {
		r.p[i] = p[i] - tol
		r.q[i] = p[i] + tol
	}
	return &r
}

func initBoundingBox(r, r1, r2 *Rect) {
	*r = *r1
	r.enlarge(r2)
}

// boundingBox constructs the smallest rectangle containing both r1 and r2.
func boundingBox(r1, r2 *Rect) *Rect {
	var r Rect
	initBoundingBox(&r, r1, r2)
	return &r
}

// boundingBoxN constructs the smallest rectangle containing all of r...
func boundingBoxN(rects ...*Rect) (bb *Rect) {
	if len(rects) == 1 {
		bb = rects[0]
		return
	}
	bb = boundingBox(rects[0], rects[1])
	for _, rect := range rects[2:] {
		bb = boundingBox(bb, rect)
	}
	return
}
