package transition

import "errors"

var (
	ErrAttestationsSignatureInvalid          = errors.New("attestations signature invalid")
	ErrRandaoSignatureInvalid                = errors.New("randao signature invalid")
	ErrBLSToExecutionChangesSignatureInvalid = errors.New("BLS to execution changes signature invalid")
)
