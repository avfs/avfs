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
	"encoding/gob"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/avfs/avfs"
)

type PermTest struct {
	sfs     *SuiteFS
	testMap testMap
	name    string
	path    string
	permDir string
	create  bool
}

// testMap stores current permission tests where the key is composed of userName and mode
// and the value is an error.
type testMap map[string]error

// PermFunc returns an error depending on the permissions of the user and the file mode on the path.
type PermFunc func(path string) error

// NewPermTest loads a golden file.
func NewPermTest(sfs *SuiteFS, testDir string) *PermTest {
	name := filepath.Base(testDir)
	fileName := fmt.Sprintf("perm%s.golden", name)
	path := filepath.Join(sfs.initDir, "testdata", fileName)

	pt := &PermTest{
		sfs:     sfs,
		testMap: make(testMap),
		name:    name,
		path:    path,
		create:  false,
	}

	pt.permDir = filepath.Join(testDir, pt.PermFolder())

	return pt
}

func (pt *PermTest) PermFolder() string {
	return "perm"
}

// CreateDirs creates directories and sets permissions to all tests directories.
func (pt *PermTest) CreateDirs(t *testing.T) {
	sfs := pt.sfs
	vfs := sfs.vfsSetup
	sfs.User(t, UsrTest)

	for _, ui := range UserInfos() {
		usrDir := vfs.Join(pt.permDir, ui.Name)
		sfs.CreateDir(t, usrDir, 0o777)

		for m := fs.FileMode(0); m <= 0o777; m++ {
			permDir := vfs.Join(usrDir, m.String())
			sfs.CreateDir(t, permDir, m)
		}
	}
}

func (pt *PermTest) registerGob() {
	gob.Register(&fs.PathError{})
	gob.Register(&os.LinkError{})
	gob.Register(syscall.ENOENT)
}

// Load loads a permissions test file.
func (pt *PermTest) Load(t *testing.T) {
	f, err := os.Open(pt.path)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("Open %s : want error to be nil, got %v", pt.path, err)
		}

		pt.create = true

		return
	}

	defer f.Close()

	gd := gob.NewDecoder(f)

	pt.registerGob()

	err = gd.Decode(&pt.testMap)
	if err != nil {
		t.Fatalf("Decode error : %v", err)
	}
}

// Save saves a permissions test file.
func (pt *PermTest) Save(t *testing.T) {
	if !pt.create {
		return
	}

	sfs := pt.sfs
	sfs.User(t, sfs.initUser.Name())

	f, err := os.Create(pt.path)
	if err != nil {
		t.Fatalf("Create %s : want error to be nil, got %v", pt.path, err)
	}

	defer f.Close()

	ge := gob.NewEncoder(f)

	pt.registerGob()

	err = ge.Encode(pt.testMap)
	if err != nil {
		t.Fatalf("Encode error : %v", err)
	}
}

func (pt *PermTest) NormalizeError(err error) error {
	switch e := err.(type) {
	case *fs.PathError:
		return &fs.PathError{Op: e.Op, Path: strings.TrimPrefix(e.Path, pt.permDir), Err: e.Err}
	case *os.LinkError:
		return &os.LinkError{
			Op:  e.Op,
			Old: strings.TrimPrefix(e.Old, pt.permDir),
			New: strings.TrimPrefix(e.New, pt.permDir),
			Err: e.Err,
		}
	default:
		return e
	}
}

// CompareErrors compares wanted error to error.
func (pt *PermTest) CompareErrors(t *testing.T, wantErr, err error) {
	if reflect.DeepEqual(wantErr, err) { //nolint:govet // avoid using reflect.DeepEqual with errors.
		return
	}

	if wantErr.Error() != err.Error() {
		t.Errorf("want error to be %v, got %v", wantErr, err)
	}
}

// Test generates or tests the golden file of the permissions for a specific function.
func (pt *PermTest) Test(t *testing.T, permFunc PermFunc) {
	sfs := pt.sfs
	vfs := sfs.vfsSetup

	pt.Load(t)

	if pt.create && !vfs.HasFeature(avfs.FeatRealFS) {
		t.Errorf("Can't test emulated file system %s : golden file not present", vfs.Type())

		return
	}

	pt.CreateDirs(t)

	for _, ui := range UserInfos() {
		u := sfs.User(t, ui.Name)
		for m := fs.FileMode(0); m <= 0o777; m++ {
			usrMode := fmt.Sprintf("%s/%s", u.Name(), m)
			permDir := vfs.Join(pt.permDir, usrMode)
			err := permFunc(permDir)

			e := pt.NormalizeError(err)
			if pt.create {
				pt.testMap[usrMode] = e

				continue
			}

			wantErr, ok := pt.testMap[usrMode]
			if !ok {
				t.Fatalf("%s : cant find result for %s", pt.name, usrMode)
			}

			pt.CompareErrors(t, wantErr, e)
		}
	}

	pt.Save(t)
}
