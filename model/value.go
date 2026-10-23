// Copyright 2015 jmrozanec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Ported to Go from com/cronutils/model/field/value/{FieldValue,IntegerFieldValue,SpecialChar,
// SpecialCharFieldValue}.java.

// Package model holds the cron model: field values and expressions, field constraints and definitions, cron
// definitions with the predefined dialects, parsed crons (single, composite and @reboot) and their validation.
package model

import "strconv"

// SpecialChar is a non-numeric marker a field value can carry.
type SpecialChar int

// The special characters, in declaration order.
const (
	LW SpecialChar = iota
	L
	W
	Hash
	QuestionMarkChar
	Tilde
	None
)

var specialCharNames = [...]string{"LW", "L", "W", "HASH", "QUESTION_MARK", "TILDE", "NONE"}

// SpecialChars lists every special character in declaration order.
var SpecialChars = []SpecialChar{LW, L, W, Hash, QuestionMarkChar, Tilde, None}

// String returns the special character's name (L, LW, W, HASH, QUESTION_MARK, TILDE, NONE).
func (s SpecialChar) String() string {
	return specialCharNames[s]
}

// FieldValue is the value of a field: an integer or a special character.
type FieldValue interface {
	// String returns the value as text: the integer in decimal, or the special character's name.
	String() string
	fieldValue()
}

// IntegerFieldValue is an integer field value.
type IntegerFieldValue int

// Value returns the integer.
func (v IntegerFieldValue) Value() int { return int(v) }

func (v IntegerFieldValue) String() string { return strconv.Itoa(int(v)) }

func (IntegerFieldValue) fieldValue() {}

// SpecialCharFieldValue is a special-character field value.
type SpecialCharFieldValue SpecialChar

// Value returns the special character.
func (v SpecialCharFieldValue) Value() SpecialChar { return SpecialChar(v) }

func (v SpecialCharFieldValue) String() string { return SpecialChar(v).String() }

func (SpecialCharFieldValue) fieldValue() {}
