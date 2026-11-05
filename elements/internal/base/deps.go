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
			if x != y {
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
