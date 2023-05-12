//
//  Copyright 2021 The AVFS authors
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
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avfs/avfs"
)

// PermTests regroups all tests for a specific function.
type PermTests struct {
	sfs           *SuiteFS              // sfs is a test suite for virtual file systems.
	errors        map[string]*permError // errors store the errors for each "User/Permission" combination.
	errFileName   string                // errFileName is the json file storing the test results from OsFs.
	permDir       string                // permDir is the root directory of the test environment.
	options       PermOptions           // PermOptions are options for running the tests.
	errFileExists bool                  // errFileExists indicates if errFileName exits.
}

// errType defines the error type.
type errType string

const (
	LinkError   errType = "LinkError"
	PathError   errType = "PathError"
	StringError errType = "StringError"
)

// permError is the error returned by each test, to be stored as json.
type permError struct {
	ErrType errType `json:"errType,omitempty"`
	ErrOp   string  `json:"errOp,omitempty"`
	ErrPath string  `json:"errPath,omitempty"`
	ErrOld  string  `json:"errOld,omitempty"`
	ErrNew  string  `json:"errNew,omitempty"`
	ErrErr  string  `json:"errErr,omitempty"`
}

// PermOptions are options for running the tests.
type PermOptions struct {
	IgnoreOp    bool // IgnoreOp ignores the Op field comparison of fs.PathError or os.LinkError structs.
	IgnorePath  bool // IgnorePath ignores the Path, Old or New field comparison of fs.PathError or os.LinkError errors.
	CreateFiles bool // CreateFiles creates files instead of directories.
}

// NewPermTests creates and returns a new environment for permissions test.
func (sfs *SuiteFS) NewPermTests(t *testing.T, testDir, funcName string) *PermTests {
	return sfs.NewPermTestsWithOptions(t, testDir, funcName, &PermOptions{})
}

// NewPermTestsWithOptions creates and returns a new environment for permissions test with options.
func (sfs *SuiteFS) NewPermTestsWithOptions(t *testing.T, testDir, funcName string, options *PermOptions) *PermTests {
	osName := avfs.CurrentOSType().String()
	errFileName := filepath.Join(sfs.initDir, "testdata", fmt.Sprintf("perm%s%s.golden", funcName, osName))
	permDir := filepath.Join(testDir, funcName)

	pts := &PermTests{
		sfs:           sfs,
		errors:        make(map[string]*permError),
		errFileName:   errFileName,
		errFileExists: true,
		permDir:       permDir,
		options:       *options,
	}

	vfs := sfs.vfsSetup
	adminUser := vfs.Idm().AdminUser()

	sfs.setUser(t, adminUser.Name())
	sfs.createDir(t, pts.permDir, avfs.DefaultDirPerm)

	for _, ui := range UserInfos() {
		sfs.setUser(t, ui.Name)

		usrDir := vfs.Join(pts.permDir, ui.Name)
		sfs.createDir(t, usrDir, avfs.DefaultDirPerm)

		for m := fs.FileMode(0); m <= 0o777; m++ {
			path := vfs.Join(usrDir, m.String())
			if pts.options.CreateFiles {
				sfs.createFile(t, path, m)
			} else {
				sfs.createDir(t, path, m)
			}
		}

		// Allow updates from user and group.
		err := vfs.Chmod(usrDir, 0o775)
		RequireNoError(t, err, "Chmod %s", usrDir)
	}

	sfs.setUser(t, UsrTest)

	return pts
}

// PermFunc returns an error depending on the permissions of the user and the file mode on the path.
type PermFunc func(path string) error

// load loads a permissions test file.
func (pts *PermTests) load(t *testing.T) {
	sfs := pts.sfs
	sfs.setUser(t, sfs.initUser.Name())

	b, err := os.ReadFile(pts.errFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			pts.errFileExists = false

			return
		}

		t.Fatalf("ReadFile %s : %v", pts.errFileName, err)
	}

	err = json.Unmarshal(b, &pts.errors)
	if err != nil {
		t.Fatalf("Unmarshal %s : %v", pts.errFileName, err)
	}
}

// save saves a permissions test file.
func (pts *PermTests) save(t *testing.T) {
	if pts.errFileExists {
		return
	}

	b, err := json.MarshalIndent(pts.errors, "", "\t")
	if err != nil {
		t.Fatalf("MarshalIndent %s : %v", pts.errFileName, err)
	}

	sfs := pts.sfs
	sfs.setUser(t, sfs.initUser.Name())

	err = os.WriteFile(pts.errFileName, b, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("ReadFile %s : %v", pts.errFileName, err)
	}
}

// newPermError creates and returns a normalized permError where all paths are relative to permDir.
func (pts *PermTests) newPermError(err error) *permError {
	prefix := pts.permDir + string(os.PathSeparator)

	switch e := err.(type) {
	case *fs.PathError:
		return &permError{
			ErrType: PathError,
			ErrOp:   e.Op,
			ErrPath: strings.TrimPrefix(e.Path, prefix),
			ErrErr:  e.Err.Error(),
		}

	case *os.LinkError:
		return &permError{
			ErrType: LinkError,
			ErrOp:   e.Op,
			ErrOld:  strings.TrimPrefix(e.Old, prefix),
			ErrNew:  strings.TrimPrefix(e.New, prefix),
			ErrErr:  e.Err.Error(),
		}
	case nil:
		return &permError{}
	default:
		return &permError{
			ErrType: StringError,
			ErrErr:  e.Error(),
		}
	}
}

// Test generates or tests the golden file of the permissions for a specific function.
func (pts *PermTests) Test(t *testing.T, permFunc PermFunc) {
	sfs := pts.sfs
	vfs := sfs.vfsSetup

	pts.load(t)

	if !pts.errFileExists && !vfs.HasFeature(avfs.FeatRealFS) {
		t.Errorf("Can't test emulated file system %s before a real file system.", vfs.Type())

		return
	}

	sfs.setUser(t, UsrTest)

	for _, ui := range UserInfos() {
		for m := fs.FileMode(0); m <= 0o777; m++ {
			relPath := vfs.Join(ui.Name, m.String())

			path := vfs.Join(pts.permDir, relPath)
			err := permFunc(path)
			pe := pts.newPermError(err)

			if pts.errFileExists {
				wantErr, ok := pts.errors[relPath]
				if !ok {
					t.Fatalf("Compare %s : no test recorded", path)
				}

				errStr := pts.compare(wantErr, pe)
				if errStr != "" {
					t.Errorf("Compare %s : %s", relPath, errStr)
				}
			} else {
				pts.errors[relPath] = pe
			}
		}
	}

	pts.save(t)
}

// compare compares wanted error to error and returns a non-empty string if there is an error.
func (pts *PermTests) compare(wantErr, err *permError) string {
	po := pts.options
	errStr := ""

	if err.ErrType != wantErr.ErrType {
		errStr += fmt.Sprintf("\n\twant error type to be %s, got %s", wantErr.ErrType, err.ErrType)
	}

	if !po.IgnoreOp && (wantErr.ErrType == PathError || wantErr.ErrType == LinkError) && err.ErrOp != wantErr.ErrOp {
		errStr += fmt.Sprintf("\n\twant Op to be %s, got %s", wantErr.ErrOp, err.ErrOp)
	}

	if !po.IgnorePath {
		if wantErr.ErrType == PathError && err.ErrPath != wantErr.ErrPath {
			errStr += fmt.Sprintf("\n\twant path to be %s, got %s", wantErr.ErrPath, err.ErrPath)
		}

		if wantErr.ErrType == LinkError {
			if err.ErrOld != wantErr.ErrOld {
				errStr += fmt.Sprintf("\n\twant Old to be %s, got %s", wantErr.ErrOld, err.ErrOld)
			}

			if err.ErrNew != wantErr.ErrNew {
				errStr += fmt.Sprintf("\n\twant New to be %s, got %s", wantErr.ErrNew, err.ErrNew)
			}
		}
	}

	if err.ErrErr != wantErr.ErrErr {
		errStr += fmt.Sprintf("\n\twant error to be %s, got %s", wantErr.ErrErr, err.ErrErr)
	}

	return errStr
}
