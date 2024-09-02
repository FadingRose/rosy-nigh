package fuzz

func collectVals[T comparable, U any](m map[T]U) []U {
	var res []U
	for _, v := range m {
		res = append(res, v)
	}
	return res
}

func containsAny[T any](s []T, e T, eq func(a, b T) bool) bool {
	for _, a := range s {
		if eq(a, e) {
			return true
		}
	}
	return false
}

func contains[T comparable](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func removeAny[T any](ts []T, t T, eq func(T, T) bool) []T {
	for i, v := range ts {
		if eq(v, t) {
			ts = append(ts[:i], ts[i+1:]...)
		}
	}
	return ts
}

func remove[T comparable](ts []T, t T) []T {
	for i, v := range ts {
		if v == t {
			ts = append(ts[:i], ts[i+1:]...)
		}
	}
	return ts
}

func refine[T any](ts []T, f func(T) bool) []T {
	var res []T
	for _, v := range ts {
		if !f(v) {
			res = append(res, v)
		}
	}
	return res
}
