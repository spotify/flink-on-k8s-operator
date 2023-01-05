package util

import "reflect"

type DiffValue struct {
	Left  any
	Right any
}

func MapDiff[K comparable, V any](a, b map[K]V) map[K]DiffValue {
	c := make(map[K]DiffValue)
	for k, v := range a {
		if bv, ok := b[k]; ok {
			if reflect.DeepEqual(v, bv) {
				continue
			} else {
				c[k] = DiffValue{v, bv}
			}
		}
	}
	return c
}
