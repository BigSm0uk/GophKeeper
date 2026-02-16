package validation

import (
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Validator struct {
	errors []error
}

func New() *Validator {
	return &Validator{}
}

// Check adds an error if the condition is false.
func (v *Validator) Check(ok bool, field, msg string) {
	if !ok {
		v.Add(field, msg)
	}
}

// Add adds a specific error message for a field.
func (v *Validator) Add(field, msg string) {
	v.errors = append(v.errors, fmt.Errorf("%s: %s", field, msg))
}

// Err returns the accumulated error as a gRPC status error, or nil.
func (v *Validator) Err() error {
	if len(v.errors) == 0 {
		return nil
	}

	_ = errors.Join(v.errors...)

	msgs := make([]string, len(v.errors))
	for i, e := range v.errors {
		msgs[i] = e.Error()
	}

	return status.Error(codes.InvalidArgument, strings.Join(msgs, "; "))
}
