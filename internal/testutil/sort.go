package testutil

import "sort"

func Sort[T any](s []T, predicate func(a, b T) bool) []T {
	sCopy := make([]T, len(s))
	copy(sCopy, s)

	sort.Slice(sCopy, func(i, j int) bool { return predicate(sCopy[i], sCopy[j]) })
	return sCopy
}
