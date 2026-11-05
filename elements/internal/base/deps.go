package base

import "reflect"

func depsEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x == nil || y == nil {
			if x != y {
				return false
			}
			continue
		}
		tx, ty := reflect.TypeOf(x), reflect.TypeOf(y)
		if tx != ty {
			return false
		}
		if tx.Comparable() {
			if !safeEqual(x, y) {
				return false
			}
			continue
		}
		if !reflect.DeepEqual(x, y) {
			return false
		}
	}
	return true
}

// safeEqual is x == y for values of a comparable type, which still
// panics when an interface inside holds an uncomparable value (a struct
// with an interface field, say): that counts as a change.
func safeEqual(x, y any) (eq bool) {
	defer func() {
		if recover() != nil {
			eq = false
		}
	}()
	return x == y
}
