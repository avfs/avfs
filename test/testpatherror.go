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
	"io/fs"
	"reflect"
	"testing"

	"github.com/avfs/avfs"
)

// assertPathError stores the current fs.PathError test data.
type assertPathError struct {
	tb   testing.TB
	err  error
	op   string
	path string
}

// AssertPathError checks if err is a fs.PathError.
func AssertPathError(tb testing.TB, err error) *assertPathError {
	tb.Helper()

	if err == nil {
		tb.Error("want error to be not nil, got nil")

		return &assertPathError{tb: tb, op: "", path: "", err: nil}
	}

	e, ok := err.(*fs.PathError)
	if !ok {
		tb.Errorf("want error type to be *fs.PathError, got %v : %v", reflect.TypeOf(err), err)

		return &assertPathError{tb: tb, op: "", path: "", err: nil}
	}

	return &assertPathError{tb: tb, op: e.Op, path: e.Path, err: e.Err}
}

// Op checks if op is equal to the current fs.PathError Op for one of the osTypes.
func (cp *assertPathError) Op(op string, osTypes ...avfs.OSType) *assertPathError {
	tb := cp.tb
	tb.Helper()

	if cp.err == nil {
		return cp
	}

	canTest := len(osTypes) == 0

	for _, osType := range osTypes {
		if osType == avfs.CurrentOSType() {
			canTest = true
		}
	}

	if !canTest {
		return cp
	}

	if cp.op != op {
		tb.Errorf("want Op to be %s, got %s", op, cp.op)
	}

	return cp
}

// OpStat checks if the current fs.PathError Op is a Stat Op.
func (cp *assertPathError) OpStat() *assertPathError {
	tb := cp.tb
	tb.Helper()

	return cp.
		Op("stat", avfs.OsLinux).
		Op("CreateFile", avfs.OsWindows)
}

// OpLstat checks if the current fs.PathError Op is a Lstat Op.
func (cp *assertPathError) OpLstat() *assertPathError {
	tb := cp.tb
	tb.Helper()

	return cp.
		Op("lstat", avfs.OsLinux).
		Op("CreateFile", avfs.OsWindows)
}

// Path checks the path of the current fs.PathError.
func (cp *assertPathError) Path(path string) *assertPathError {
	tb := cp.tb
	tb.Helper()

	if cp.err == nil {
		return cp
	}

	if cp.path != path {
		tb.Errorf("want Path to be %s, got %s", path, cp.path)
	}

	return cp
}

// Err checks the error of current fs.PathError.
func (cp *assertPathError) Err(wantErr error, osTypes ...avfs.OSType) *assertPathError {
	tb := cp.tb
	tb.Helper()

	if cp.err == nil {
		return cp
	}

	canTest := len(osTypes) == 0

	for _, osType := range osTypes {
		if osType == avfs.CurrentOSType() {
			canTest = true

			break
		}
	}

	if !canTest {
		return cp
	}

	e := reflect.ValueOf(cp.err)
	we := reflect.ValueOf(wantErr)

	if e.CanUint() && we.CanUint() && e.Uint() == we.Uint() {
		return cp
	}

	if cp.err.Error() == wantErr.Error() {
		return cp
	}

	tb.Errorf("%s : want error to be %v, got %v", cp.path, wantErr, cp.err)

	return cp
}

// ErrPermDenied checks if the current fs.PathError is a permission denied error.
func (cp *assertPathError) ErrPermDenied() *assertPathError {
	cp.tb.Helper()

	return cp.
		Err(avfs.ErrPermDenied, avfs.OsLinux).
		Err(avfs.ErrWinAccessDenied, avfs.OsWindows)
}
