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
	gotErr       error       // gotErr is the error to test.
	wantErr      error       // wantErr is the expected Err of the fs.PathError or the os.LinkError.
	wantPath     string      // wantPath is the expected Path for a fs.PathError.
	wantOld      string      // wantOld is the expected Old for an os.LinkError.
	wantNew      string      // wantNew is the expected New for an os.LinkError.
	wantOp       string      // wantOp is the expected Op for the error.
	wantMinGoVer int         // wantMinGoVer is the minimum go version returned by goVerNum function, 0 if not defined.
	wantMaxGoVer int         // wantMaxGoVer is the minimum go version returned by goVerNum function, math.MaxInt if not defined.
	wantOsType   avfs.OSType // wantOsType is the OS wanted to perform the test.
	isLinkError  bool        // isLinkError is true if the expected error is an os.LinkError.
	testOp       bool        // testOp indicates if the Op field should be tested.
	testPath     bool        // testPath indicates if the Path field should be tested.
	testOld      bool        // testOld indicates if the Old field should be tested.
	testNew      bool        // testNew indicates if the New field should be tested.
}

func assertPLError(tb testing.TB, err error, isLinkError bool) *assertError {
	return &assertError{
		tb:           tb,
		gotErr:       err,
		wantMaxGoVer: math.MaxInt,
		wantOsType:   avfs.CurrentOSType(),
		isLinkError:  isLinkError,
	}
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
	ae.testNew = true
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
	ae.testOld = true
	ae.wantOld = WantOld

	return ae
}

// Op sets the expected Op.
func (ae *assertError) Op(op string) *assertError {
	ae.testOp = true
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
	ae.testPath = true
	ae.wantPath = path

	return ae
}

func (ae *assertError) Test() *assertError {
	ae.tb.Helper()

	canTest := goVersion >= ae.wantMinGoVer && goVersion <= ae.wantMaxGoVer && ae.wantOsType == avfs.CurrentOSType()
	if !canTest {
		return ae
	}

	if ae.wantErr == nil {
		if ae.gotErr != nil {
			ae.errorf("want error to be nil, got %v", ae.gotErr)
		}

		return ae
	}

	var (
		op  string
		err error
	)

	switch e := ae.gotErr.(type) {
	case *fs.PathError:
		op, err = e.Op, e.Err

		if ae.testPath && ae.wantPath != e.Path {
			ae.errorf("want Path to be %s, got %s", ae.wantPath, e.Path)
		}
	case *os.LinkError:
		op, err = e.Op, e.Err

		if ae.testOld && ae.wantOld != e.Old {
			ae.errorf("want Old to be %s, got %s", ae.wantOld, e.Old)
		}

		if ae.testNew && ae.wantNew != e.New {
			ae.errorf("want New to be %s, got %s", ae.wantNew, e.New)
		}
	default:
		ae.errorf("want error type to be %v, got %v", reflect.TypeOf(ae.wantErr), reflect.TypeOf(ae.gotErr))

		return ae
	}

	if ae.testOp && ae.wantOp != op {
		ae.errorf("want Op to be %s, got %s", ae.wantOp, op)
	}

	we, e := reflect.ValueOf(ae.wantErr), reflect.ValueOf(err)
	if we.CanUint() && e.CanUint() {
		weu, eu := we.Uint(), e.Uint()
		if weu != eu {
			ae.errorf("want error to be %s (0x%X), got %s (0x%X)", ae.wantErr, weu, err, eu)
		}

		return ae
	}

	if ae.wantErr.Error() != err.Error() {
		ae.errorf("want error to be %s, got %s", ae.wantErr, err)
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

// ErrFileNotFound sets the expected error when a file is not found for the current OS.
func (ae *assertError) ErrFileNotFound() *assertError {
	switch avfs.CurrentOSType() {
	case avfs.OsWindows:
		ae.Err(avfs.ErrWinFileNotFound)
	default:
		ae.Err(avfs.ErrNoSuchFileOrDir)
	}

	return ae
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
