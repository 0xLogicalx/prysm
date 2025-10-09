package state

// WriteOnlyGloas defines a struct which only has write access to Gloas field methods.
type WriteOnlyGloas interface {
	UpdateExecutionPayloadAvailabilityAtIndex(idx uint64, val byte) error
}
