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

const (
	LinkError   = "LinkError"
	PathError   = "PathError"
	StringError = "StringError"
)

type PermTests struct {
	sfs      *SuiteFS
	tests    map[string]*permTest
	options  *PermOptions
	permDir  string
	testFile string
	create   bool
}

type permTest struct {
	ErrType string `json:"errType,omitempty"`
	ErrOp   string `json:"errOp,omitempty"`
	ErrPath string `json:"errPath,omitempty"`
	ErrOld  string `json:"errOld,omitempty"`
	ErrNew  string `json:"errNew,omitempty"`
	ErrErr  string `json:"errErr,omitempty"`
}

type PermOptions struct {
	IgnoreOp   bool
	IgnorePath bool
}

var PermDefaultOptions = &PermOptions{} //nolint:gochecknoglobals // PermDefaultOptions regroups the default options.

func (sfs *SuiteFS) NewPermTests(testDir, funcName string, options *PermOptions) *PermTests {
	osName := avfs.CurrentOSType().String()
	testFile := filepath.Join(sfs.initDir, "testdata", fmt.Sprintf("perm%s%s.golden", funcName, osName))
	permDir := filepath.Join(testDir, "perm")

	pts := &PermTests{
		sfs:      sfs,
		tests:    make(map[string]*permTest),
		options:  options,
		permDir:  permDir,
		testFile: testFile,
		create:   false,
	}

	return pts
}

// PermFunc returns an error depending on the permissions of the user and the file mode on the path.
type PermFunc func(path string) error

// CreateDirs creates directories and sets permissions to all tests directories.
func (pts *PermTests) CreateDirs(t *testing.T) {
	sfs := pts.sfs
	vfs := sfs.vfsSetup
	adminUser := vfs.Idm().AdminUser()

	sfs.SetUser(t, adminUser.Name())
	sfs.CreateDir(t, pts.permDir, 0o777)

	for _, ui := range UserInfos() {
		sfs.SetUser(t, ui.Name)

		usrDir := vfs.Join(pts.permDir, ui.Name)
		sfs.CreateDir(t, usrDir, 0o777)

		for m := fs.FileMode(0); m <= 0o777; m++ {
			usrModDir := vfs.Join(usrDir, m.String())
			sfs.CreateDir(t, usrModDir, m)
		}

		// Allow updates from user and group.
		err := vfs.Chmod(usrDir, 0o775)
		CheckNoError(t, "Chmod "+usrDir, err)
	}
}

// Load loads a permissions test file.
func (pts *PermTests) load(t *testing.T) {
	b, err := os.ReadFile(pts.testFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			pts.create = true

			return
		}

		t.Fatalf("ReadFile %s : %v", pts.testFile, err)
	}

	err = json.Unmarshal(b, &pts.tests)
	if err != nil {
		t.Fatalf("Unmarshal %s : %v", pts.testFile, err)
	}
}

// Save saves a permissions test file.
func (pts *PermTests) save(t *testing.T) {
	if !pts.create {
		return
	}

	sfs := pts.sfs
	sfs.SetUser(t, sfs.initUser.Name())

	b, err := json.MarshalIndent(pts.tests, "", "\t")
	if err != nil {
		t.Fatalf("MarshalIndent %s : %v", pts.testFile, err)
	}

	err = os.WriteFile(pts.testFile, b, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("ReadFile %s : %v", pts.testFile, err)
	}
}

func (pts *PermTests) addTest(path string, err error) bool {
	if pts.create {
		pt := pts.newPermTest(err)
		pts.tests[path] = pt
	}

	return pts.create
}

func (pts *PermTests) newPermTest(err error) *permTest {
	prefix := pts.permDir + string(os.PathSeparator)

	switch e := err.(type) {
	case *fs.PathError:
		return &permTest{
			ErrType: PathError,
			ErrOp:   e.Op,
			ErrPath: strings.TrimPrefix(e.Path, prefix),
			ErrErr:  e.Err.Error(),
		}

	case *os.LinkError:
		return &permTest{
			ErrType: LinkError,
			ErrOp:   e.Op,
			ErrOld:  strings.TrimPrefix(e.Old, prefix),
			ErrNew:  strings.TrimPrefix(e.New, prefix),
			ErrErr:  e.Err.Error(),
		}
	case nil:
		return &permTest{}
	default:
		return &permTest{
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

	if pts.create && !vfs.HasFeature(avfs.FeatRealFS) {
		t.Errorf("Can't test emulated file system %s before a real file system.", vfs.Type())

		return
	}

	sfs.SetUser(t, UsrTest)

	for _, ui := range UserInfos() {
		for m := fs.FileMode(0); m <= 0o777; m++ {
			relPath := vfs.Join(ui.Name, m.String())
			path := vfs.Join(pts.permDir, relPath)

			err := permFunc(path)
			if pts.addTest(relPath, err) {
				continue
			}

			pts.compare(t, relPath, err)
		}
	}

	pts.save(t)
}

// compare compares wanted error to error.
func (pts *PermTests) compare(t *testing.T, path string, err error) {
	wantPt, ok := pts.tests[path]
	if !ok {
		t.Fatalf("Compare %s : no test recorded", path)
	}

	var errStr string

	pt := pts.newPermTest(err)
	if pt.ErrType != wantPt.ErrType {
		errStr += fmt.Sprintf("\n\twant error type to be %s, got %s", wantPt.ErrType, pt.ErrType)
	}

	if !pts.options.IgnoreOp && (pt.ErrType == PathError || pt.ErrType == LinkError) && pt.ErrOp != wantPt.ErrOp {
		errStr += fmt.Sprintf("\n\twant Op to be %s, got %s", wantPt.ErrOp, pt.ErrOp)
	}

	if !pts.options.IgnorePath {
		if pt.ErrType == PathError && pt.ErrPath != wantPt.ErrPath {
			errStr += fmt.Sprintf("\n\twant path to be %s, got %s", wantPt.ErrPath, pt.ErrPath)
		}

		if pt.ErrType == LinkError {
			if pt.ErrOld != wantPt.ErrOld {
				errStr += fmt.Sprintf("\n\twant Old to be %s, got %s", wantPt.ErrOld, pt.ErrOld)
			}

			if pt.ErrNew != wantPt.ErrNew {
				errStr += fmt.Sprintf("\n\twant New to be %s, got %s", wantPt.ErrNew, pt.ErrNew)
			}
		}
	}

	if pt.ErrErr != wantPt.ErrErr {
		errStr += fmt.Sprintf("\n\twant error to be %s, got %s", wantPt.ErrErr, pt.ErrErr)
	}

	if errStr != "" {
		t.Errorf("Compare %s : %s", path, errStr)
	}
}
