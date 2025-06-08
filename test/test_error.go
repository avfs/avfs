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
	"math"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"testing"

	"github.com/avfs/avfs"
)

var goVersion = goVerNum(runtime.Version()) //nolint:gochecknoglobals // Stores the current Go version

// assertError stores the current fs.PathError or the os.LinkError test data.
type assertError struct {
	tb           testing.TB  // tb is the interface common to T, B, and F for the current test.
	err          error       // err is the error to test.
	wantOsType   avfs.OSType // wantOsType is the OS
	wantMinGoVer int         // wantMinGoVer is the minimum go version returned by goVerNum function, 0 if not defined.
	wantMaxGoVer int         // wantMaxGoVer is the minimum go version returned by goVerNum function, math.MaxInt if not defined.
	wantPath     string      // wantPath is the expected Path for a fs.PathError.
	wantOld      string      // wantOld is the expected Old for a os.LinkError.
	wantNew      string      // wantNew is the expected New for a os.LinkError.
	wantOp       string      // wantOp is the expected Op for the error.
	wantErr      error       // wantErr is the expected Err of the fs.PathError or the os.LinkError.
	isLinkError  bool        // isLinkError is true if the expected error is a os.LinkError.
}

func assertPLError(tb testing.TB, err error, isLinkError bool) *assertError {
	return &assertError{tb: tb, err: err, wantMaxGoVer: math.MaxInt, isLinkError: isLinkError}
}

// AssertPathError checks an error of type fs.PathError.
func AssertPathError(tb testing.TB, err error) *assertError {
	return assertPLError(tb, err, false)
}

// AssertLinkError checks an error of type os.LinkError.
func AssertLinkError(tb testing.TB, err error) *assertError {
	return assertPLError(tb, err, true)
}

// GoVersion sets the expected Go version.
func (ae *assertError) GoVersion(minGoVer, maxGoVer string) *assertError {
	ae.wantMinGoVer = goVerNum(minGoVer)
	ae.wantMaxGoVer = goVerNum(maxGoVer)

	if ae.wantMaxGoVer == 0 {
		ae.wantMaxGoVer = math.MaxInt
	}

	return ae
}

// goVerNum returns a comparable Go version number derived from major.minor version string.
func goVerNum(version string) int {
	re := regexp.MustCompile(`\d+`)

	v := re.FindAllString(version, 2)
	if len(v) != 2 {
		return 0
	}

	vMaj, err := strconv.Atoi(v[0])
	if err != nil {
		return 0
	}

	vMin, err := strconv.Atoi(v[1])
	if err != nil {
		return 0
	}

	return vMaj<<16 + vMin
}

// Err sets the expected error.
func (ae *assertError) Err(wantErr error) *assertError {
	ae.wantErr = wantErr

	return ae
}

// New sets the expected new path.
func (ae *assertError) New(wantNew string) *assertError {
	ae.wantNew = wantNew

	return ae
}

// NoError set the expected error to nil.
func (ae *assertError) NoError() *assertError {
	ae.wantErr = nil

	return ae
}

// Old sets the expected old path.
func (ae *assertError) Old(WantOld string) *assertError {
	ae.wantOld = WantOld

	return ae
}

// Op sets the expected Op.
func (ae *assertError) Op(op string) *assertError {
	ae.wantOp = op

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
func (ae *assertError) OSType(ost avfs.OSType) *assertError {
	ae.wantOsType = ost

	return ae
}

// Path sets the expected path.
func (ae *assertError) Path(path string) *assertError {
	ae.wantPath = path

	return ae
}

func (ae *assertError) Test() *assertError {
	ae.tb.Helper()

	if !ae.cantTest() {
		return ae
	}

	var (
		op  string
		err error
	)

	if e, ok := ae.err.(*fs.PathError); ok && !ae.isLinkError {
		op = e.Op
		err = e.Err

		if ae.wantPath != "" && ae.wantPath != e.Path {
			ae.errorf("want Path to be %s, got %s", ae.wantPath, e.Path)
		}
	} else if e, ok := ae.err.(*os.LinkError); ok && ae.isLinkError {
		op = e.Op
		err = e.Err

		if ae.wantOld != "" && ae.wantOld != e.Old {
			ae.errorf("want Old to be %s, got %s", ae.wantOld, e.Old)
		}

		if ae.wantNew != "" && ae.wantNew != e.New {
			ae.errorf("want New to be %s, got %s", ae.wantNew, e.New)
		}
	} else {
		ae.errorf("want error type to be %v, got %v", reflect.TypeOf(ae.wantErr), reflect.TypeOf(ae.err))

		return ae
	}

	if ae.wantOp != "" && ae.wantOp != op {
		ae.errorf("want Op to be %s, got %s", ae.wantOp, op)
	}

	wantErr := reflect.ValueOf(ae.wantErr)
	gotErr := reflect.ValueOf(err)

	if wantErr.CanUint() && gotErr.CanUint() && wantErr.Uint() != gotErr.Uint() {
		ae.errorf("want error to be %s (0x%X), got %s (0x%X)", wantErr, wantErr.Uint(), gotErr, gotErr.Uint())
	} else if ae.wantErr.Error() != err.Error() {
		ae.errorf("want error to be %s, got %s", wantErr, gotErr)
	}

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

// cantTest returns if the test can be done.
func (ae *assertError) cantTest() bool {
	if goVersion < ae.wantMinGoVer || goVersion > ae.wantMaxGoVer {
		return false
	}

	if ae.wantOsType != avfs.OsUnknown && ae.wantOsType != avfs.CurrentOSType() {
		return false
	}

	if ae.wantErr == nil {
		if ae.err != nil {
			ae.errorf("want error to be nil, got %v", ae.err)
		}

		return false
	}

	return true
}

// errorf generates an error if the Go max version is undefined, a warning otherwise.
func (ae *assertError) errorf(format string, args ...any) {
	tb := ae.tb
	tb.Helper()

	if ae.wantMaxGoVer == math.MaxInt {
		tb.Errorf(format, args...)
	} else {
		tb.Logf("WARN: "+format, args...)
	}
}
