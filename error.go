package soda

import (
	"errors"
	"fmt"
	"strings"
)

type OpenAPISpecError struct {
	Position string
	Field    string
	Reason   string
}

func (oe OpenAPISpecError) Error() string {
	return fmt.Sprintf("openapi spec error: field %s in %s is invalid, cause of %s", oe.Field, oe.Position, oe.Reason)
}

type ValidationError struct {
	Field    string `json:"field"`
	Position string `json:"in"`
	Reason   string `json:"message"`
}

func NewValidationError(position, field, reason string) *ValidationError {
	return &ValidationError{Field: field, Position: position, Reason: reason}
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error: field %q in %s is invalid, cause of %s", ve.Field, ve.Position, ve.Reason)
}

// ParseErrorKind describes a kind of ParseError.
// The type simplifies comparison of errors.
type ParseErrorKind int

const (
	// KindOther describes an untyped parsing error.
	KindOther ParseErrorKind = iota
	// KindUnsupportedFormat describes an error that happens when a value has an unsupported format.
	KindUnsupportedFormat
	// KindInvalidFormat describes an error that happens when a value does not conform a format
	// that is required by a serialization method.
	KindInvalidFormat
)

// ParseError describes errors which happens while parse operation's parameters, requestBody, or response.
type ParseError struct {
	Value  interface{}
	Cause  error
	Reason string
	path   []interface{}
	Kind   ParseErrorKind
}

func (e *ParseError) Error() string {
	var msg []string
	if p := e.Path(); len(p) > 0 {
		var arr []string
		for _, v := range p {
			arr = append(arr, fmt.Sprintf("%v", v))
		}
		msg = append(msg, fmt.Sprintf("path %v", strings.Join(arr, ".")))
	}
	msg = append(msg, e.innerError())
	return strings.Join(msg, ": ")
}

func (e *ParseError) innerError() string {
	var msg []string
	if e.Value != nil {
		msg = append(msg, fmt.Sprintf("value %v", e.Value))
	}
	if e.Reason != "" {
		msg = append(msg, e.Reason)
	}
	if e.Cause != nil {
		var v *ParseError
		if ok := errors.Is(e.Cause, v); ok {
			msg = append(msg, v.innerError())
		} else {
			msg = append(msg, e.Cause.Error())
		}
	}
	return strings.Join(msg, ": ")
}

// RootCause returns a root cause of ParseError.
func (e *ParseError) RootCause() error {
	var pe *ParseError
	if ok := errors.Is(e.Cause, pe); ok {
		return pe.RootCause()
	}
	return e.Cause
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}

// Path returns a path to the root cause.
func (e *ParseError) Path() []interface{} {
	var path []interface{}
	var v *ParseError
	if ok := errors.Is(e.Cause, v); ok {
		p := v.Path()
		if len(p) > 0 {
			path = append(path, p...)
		}
	}
	if len(e.path) > 0 {
		path = append(path, e.path...)
	}
	return path
}

type SerializationMethodError struct {
	Style   string
	Explode bool
}

func (e SerializationMethodError) Error() string {
	return fmt.Sprintf("invalid serialization method: style=%q, explode=%v", e.Style, e.Explode)
}
