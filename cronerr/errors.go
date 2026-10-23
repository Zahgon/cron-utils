// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cronerr defines the error values returned by every cron-utils package.
//
// Each error carries a Kind that tells callers what went wrong: an invalid argument (a malformed or unsupported cron
// expression), an operation that is not supported for a kind of cron, a date-time value that does not exist, a missing
// description text, or a defect inside the library (a state the computation was never meant to reach).
package cronerr

import (
	"errors"
	"fmt"
)

// Kind classifies an Error.
type Kind int

const (
	// IllegalArgument reports an invalid input, such as a cron expression that does not match its definition.
	IllegalArgument Kind = iota
	// IllegalState reports a call made on an object that is not ready for it.
	IllegalState
	// DateTime reports a date or time value that cannot exist, such as the 31st of April.
	DateTime
	// UnsupportedOperation reports an operation that a kind of cron does not support.
	UnsupportedOperation
	// MissingResource reports a description text missing from every bundle of a locale.
	MissingResource
	// NullValue reports a required value that was absent: a defect of the computation.
	NullValue
	// TypeMismatch reports a value of an unexpected type: a defect of the computation.
	TypeMismatch
	// IndexOutOfBounds reports an index outside a list: a defect of the computation.
	IndexOutOfBounds
)

var kindNames = map[Kind]string{
	IllegalArgument:      "IllegalArgumentException",
	IllegalState:         "IllegalStateException",
	DateTime:             "DateTimeException",
	UnsupportedOperation: "UnsupportedOperationException",
	MissingResource:      "MissingResourceException",
	NullValue:            "NullPointerException",
	TypeMismatch:         "ClassCastException",
	IndexOutOfBounds:     "ArrayIndexOutOfBoundsException",
}

// String returns the name under which the original library reports this kind of failure.
func (k Kind) String() string {
	return kindNames[k]
}

// Defect reports whether the kind marks a defect of the computation rather than a rejected input.
func (k Kind) Defect() bool {
	return k == NullValue || k == TypeMismatch || k == IndexOutOfBounds
}

// Error is the error type of every cron-utils package.
type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Message == "" {
		return e.Kind.String()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// New returns an error of the given kind with a formatted message.
func New(kind Kind, format string, args ...any) *Error {
	if len(args) == 0 {
		return &Error{Kind: kind, Message: format}
	}
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

// Argument returns an IllegalArgument error with a formatted message.
func Argument(format string, args ...any) *Error {
	return New(IllegalArgument, format, args...)
}

// Wrap returns an error of the given kind whose cause is err.
func Wrap(kind Kind, err error, format string, args ...any) *Error {
	e := New(kind, format, args...)
	e.Cause = err
	return e
}

// KindOf returns the kind of err and whether err is (or wraps) an *Error.
func KindOf(err error) (Kind, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind, true
	}
	return 0, false
}

// Is reports whether err is (or wraps) an *Error of the given kind.
func Is(err error, kind Kind) bool {
	k, ok := KindOf(err)
	return ok && k == kind
}
