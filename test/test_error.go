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
	"os"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"testing"

	"github.com/avfs/avfs"
)

// assertError stores the current fs.PathError or the os.LinkError test data.
type assertError struct {
	tb             testing.TB
	err            error
	wantOsTypes    []avfs.OSType
	wantGoVersions []string
	wantOps        []string
	wantPath       string
	wantOld        string
	wantNew        string
	wantErrs       []error
	IsLinkError    bool
}

// AssertPathError checks an error of type fs.PathError.
func AssertPathError(tb testing.TB, err error) *assertError {
	return &assertError{tb: tb, err: err, IsLinkError: false}
}

// AssertLinkError checks an error of type os.LinkError.
func AssertLinkError(tb testing.TB, err error) *assertError {
	return &assertError{tb: tb, err: err, IsLinkError: true}
}

// Test runs the test.
func (ae *assertError) Test() *assertError {
	tb := ae.tb
	tb.Helper()

	if len(ae.wantOsTypes) != 0 && !slices.Contains(ae.wantOsTypes, avfs.CurrentOSType()) {
		return ae
	}

	if len(ae.wantGoVersions) != 0 {
		re := regexp.MustCompile(`go\d+\.\d+`)
		version := re.FindString(runtime.Version())

		if !slices.Contains(ae.wantGoVersions, version) {
			return ae
		}
	}

	if len(ae.wantErrs) == 0 {
		if ae.err != nil {
			tb.Errorf("want error to be nil got %v", ae.err)
		}

		return ae
	}

	if ae.IsLinkError {
		return ae.testLinkError()
	}

	return ae.testPathError()
}

func (ae *assertError) testLinkError() *assertError {
	tb := ae.tb

	e, ok := ae.err.(*os.LinkError)
	if !ok {
		tb.Errorf("want error type to be *os.LinkError, got %v : %v", reflect.TypeOf(ae.err), ae.err)

		return ae
	}

	if len(ae.wantOps) != 0 && !slices.Contains(ae.wantOps, e.Op) {
		tb.Errorf("want Op to be %s, got %s", ae.wantOps, e.Op)
	}

	if ae.wantOld != "" && ae.wantOld != e.Old {
		tb.Errorf("want Old to be %s, got %s", ae.wantOld, e.Old)
	}

	if ae.wantNew != "" && ae.wantNew != e.New {
		tb.Errorf("want New to be %s, got %s", ae.wantNew, e.New)
	}

	te := reflect.ValueOf(e.Err)
	foundOk := false

	for _, wantErr := range ae.wantErrs {
		we := reflect.ValueOf(wantErr)
		if (te.CanUint() && we.CanUint() && te.Uint() == we.Uint()) || (e.Err.Error() == wantErr.Error()) {
			foundOk = true

			break
		}
	}

	if !foundOk {
		tb.Errorf("want error to be %s, got %s", ae.wantErrs, e.Err.Error())
	}

	return ae
}

func (ae *assertError) testPathError() *assertError {
	tb := ae.tb

	e, ok := ae.err.(*fs.PathError)
	if !ok {
		tb.Errorf("want error type to be *fs.PathError, got %v : %v", reflect.TypeOf(ae.err), ae.err)

		return ae
	}

	if len(ae.wantOps) != 0 && !slices.Contains(ae.wantOps, e.Op) {
		tb.Errorf("want Op to be %s, got %s", ae.wantOps, e.Op)
	}

	if ae.wantPath != "" && ae.wantPath != e.Path {
		tb.Errorf("want Path to be %s, got %s", ae.wantPath, e.Path)
	}

	te := reflect.ValueOf(e.Err)
	foundOk := false

	for _, wantErr := range ae.wantErrs {
		we := reflect.ValueOf(wantErr)
		if (te.CanUint() && we.CanUint() && te.Uint() == we.Uint()) || (e.Err.Error() == wantErr.Error()) {
			foundOk = true

			break
		}
	}

	if !foundOk {
		tb.Errorf("want error to be %s, got %s", ae.wantErrs, e.Err.Error())
	}

	return ae
}

// GoVersion sets the expected Go version.
func (ae *assertError) GoVersion(goVersions ...string) *assertError {
	ae.wantGoVersions = goVersions

	return ae
}

// New sets the expected new path.
func (ae *assertError) New(wantNew string) *assertError {
	ae.wantNew = wantNew

	return ae
}

// NoError set the expected error to nil.
func (ae *assertError) NoError() *assertError {
	ae.wantErrs = nil

	return ae
}

// Old sets the expected old path.
func (ae *assertError) Old(WantOld string) *assertError {
	ae.wantOld = WantOld

	return ae
}

// Op sets the expected Op.
func (ae *assertError) Op(ops ...string) *assertError {
	ae.wantOps = ops

	return ae
}

// OpLstat sets the expected Lstat Op for the current OS.
func (ae *assertError) OpLstat() *assertError {
	switch avfs.CurrentOSType() {
	case avfs.OsWindows:
		return ae.Op("CreateFile")
	default:
		return ae.Op("lstat")
	}
}

// OpStat sets the expected Stat Op for the current OS.
func (ae *assertError) OpStat() *assertError {
	switch avfs.CurrentOSType() {
	case avfs.OsWindows:
		return ae.Op("CreateFile")
	default:
		return ae.Op("stat")
	}
}

// OSType sets at least one expected OSType.
func (ae *assertError) OSType(ost ...avfs.OSType) *assertError {
	ae.wantOsTypes = ost

	return ae
}

// Path sets the expected path.
func (ae *assertError) Path(path string) *assertError {
	ae.wantPath = path

	return ae
}

// Err sets the expected error.
func (ae *assertError) Err(wantErr ...error) *assertError {
	ae.wantErrs = wantErr

	return ae
}

// ErrPermDenied sets the expected permission error for the current OS.
func (ae *assertError) ErrPermDenied() *assertError {
	switch avfs.CurrentOSType() {
	case avfs.OsWindows:
		ae.Err(avfs.ErrWinAccessDenied)
	default:
		ae.Err(avfs.ErrPermDenied)
	}

	return ae
}
