//
//  Copyright 2023 The AVFS authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//  	http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package test

import (
	"os"
	"reflect"
	"testing"

	"github.com/avfs/avfs"
)

// assertLinkError stores the current os.LinkError test data.
type assertLinkError struct {
	tb  testing.TB
	err error
	op  string
	old string
	new string
}

// AssertLinkError checks if err is a os.LinkError.
func AssertLinkError(tb testing.TB, err error) *assertLinkError {
	tb.Helper()

	if err == nil {
		tb.Error("want error to be not nil")

		return &assertLinkError{tb: tb, op: "", old: "", new: "", err: nil}
	}

	e, ok := err.(*os.LinkError)
	if !ok {
		tb.Errorf("want error type to be *os.LinkError, got %v : %v", reflect.TypeOf(err), err)

		return &assertLinkError{tb: tb, op: "", old: "", new: "", err: nil}
	}

	return &assertLinkError{tb: tb, op: e.Op, old: e.Old, new: e.New, err: e.Err}
}

// Op checks if wantOp is equal to the current os.LinkError Op.
func (cl *assertLinkError) Op(wantOp string, osTypes ...avfs.OSType) *assertLinkError {
	tb := cl.tb
	tb.Helper()

	if cl.err == nil {
		return cl
	}

	canTest := len(osTypes) == 0

	for _, osType := range osTypes {
		if osType == avfs.CurrentOSType() {
			canTest = true

			break
		}
	}

	if !canTest {
		return cl
	}

	if cl.op != wantOp {
		tb.Errorf("want Op to be %s, got %s", wantOp, cl.op)
	}

	return cl
}

// Old checks the old path of the current os.LinkError.
func (cl *assertLinkError) Old(wantOld string) *assertLinkError {
	tb := cl.tb
	tb.Helper()

	if cl.err == nil {
		return cl
	}

	if cl.old != wantOld {
		tb.Errorf("want old path to be %s, got %s", wantOld, cl.old)
	}

	return cl
}

// New checks the new path of the current os.LinkError.
func (cl *assertLinkError) New(wantNew string) *assertLinkError {
	tb := cl.tb
	tb.Helper()

	if cl.err == nil {
		return cl
	}

	if cl.new != wantNew {
		tb.Errorf("want new path to be %s, got %s", wantNew, cl.new)
	}

	return cl
}

// Err checks the error of current os.LinkError.
func (cl *assertLinkError) Err(wantErr error, osTypes ...avfs.OSType) *assertLinkError {
	tb := cl.tb
	tb.Helper()

	if cl.err == nil {
		return cl
	}

	canTest := len(osTypes) == 0

	for _, osType := range osTypes {
		if osType == avfs.CurrentOSType() {
			canTest = true

			break
		}
	}

	if !canTest {
		return cl
	}

	e := reflect.ValueOf(cl.err)
	we := reflect.ValueOf(wantErr)

	if e.CanUint() && we.CanUint() && e.Uint() == we.Uint() {
		return cl
	}

	if cl.err.Error() == wantErr.Error() {
		return cl
	}

	tb.Errorf("%s %s : want error to be %v, got %v", cl.old, cl.new, wantErr, cl.err)

	return cl
}

// ErrPermDenied checks if the current os.LinkError is a permission denied error.
func (cl *assertLinkError) ErrPermDenied() *assertLinkError {
	cl.tb.Helper()

	return cl.
		Err(avfs.ErrPermDenied, avfs.OsLinux).
		Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
}
