package pointer

// Of returns a pointer to the given value.
func Of[T any](v T) *T {
	return &v
}

// ValueOrDefault dereferences p and returns the value, or def if p is nil.
func ValueOrDefault[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}
