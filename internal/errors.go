package internal

import "errors"

var (
	ErrIncompleteMetadata = errors.New("incomplete metadata")
	ErrMultiValuedTags    = errors.New("found multi-valued tags")
)
