package pluggable

import "errors"

var (
	SortedError = errors.New("Sorted")
	Initialized = errors.New("Initialized")
)
