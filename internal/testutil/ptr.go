package testutil

func Ptr[T any](v T) *T {
	return new(v)
}
