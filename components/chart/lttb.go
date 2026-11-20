package chart

import "math"

// downsample keeps n of points with the largest-triangle-three-buckets
// method: the line's shape, peaks and dips stay. Points are in x order.
func downsample(pts []Point, n int) []Point {
	if n >= len(pts) || n < 3 {
		return pts
	}
	out := make([]Point, 0, n)
	out = append(out, pts[0])
	every := float64(len(pts)-2) / float64(n-2)
	a := 0
	for i := 0; i < n-2; i++ {
		// The average of the next bucket.
		lo := int(math.Floor(float64(i+1)*every)) + 1
		hi := min(int(math.Floor(float64(i+2)*every))+1, len(pts))
		var ax, ay float64
		for _, p := range pts[lo:hi] {
			ax += p.X
			ay += p.Y
		}
		cnt := float64(max(hi-lo, 1))
		ax /= cnt
		ay /= cnt
		// The point of this bucket making the largest triangle.
		from := int(math.Floor(float64(i)*every)) + 1
		to := int(math.Floor(float64(i+1)*every)) + 1
		best, area := from, -1.0
		pa := pts[a]
		for j := from; j < to && j < len(pts); j++ {
			ar := math.Abs((pa.X-ax)*(pts[j].Y-pa.Y) - (pa.X-pts[j].X)*(ay-pa.Y))
			if ar > area {
				best, area = j, ar
			}
		}
		out = append(out, pts[best])
		a = best
	}
	return append(out, pts[len(pts)-1])
}
