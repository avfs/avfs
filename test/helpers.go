//
//  Copyright 2020 The AVFS authors
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
	"github.com/avfs/avfs/vfsutils"
)

// CheckPathError checks errors of type os.PathError.
func CheckPathError(t *testing.T, testName, wantOp, wantPath string, wantErr, gotErr error) {
	t.Helper()

	if gotErr == nil {
		t.Fatalf("%s %s : want error to be %v, got nil", testName, wantPath, wantErr)
	}

	err, ok := gotErr.(*os.PathError)
	if !ok {
		t.Fatalf("%s %s : want error type *os.PathError, got %v", testName, wantPath, reflect.TypeOf(gotErr))
	}

	if wantOp != err.Op || wantPath != err.Path || (wantErr != err.Err && wantErr.Error() != err.Err.Error()) {
		wantPathErr := &os.PathError{Op: wantOp, Path: wantPath, Err: wantErr}
		t.Errorf("%s %s: want error to be %v, got %v", testName, wantPath, wantPathErr, gotErr)
	}
}

// CheckSyscallError checks errors of type os.SyscallError.
func CheckSyscallError(t *testing.T, testName, wantOp, wantPath string, wantErr, gotErr error) {
	t.Helper()

	if gotErr == nil {
		t.Fatalf("%s %s : want error to be %v, got nil", testName, wantPath, wantErr)
	}

	err, ok := gotErr.(*os.SyscallError)
	if !ok {
		t.Fatalf("%s %s : want error type *os.SyscallError, got %v", testName, wantPath, reflect.TypeOf(gotErr))
	}

	if err.Syscall != wantOp || err.Err != wantErr {
		t.Errorf("%s %s : want error to be %v, got %v", testName, wantPath, wantErr, err)
	}
}

// CheckLinkError checks errors of type os.LinkError.
func CheckLinkError(t *testing.T, testName, wantOp, oldPath, newPath string, wantErr, gotErr error) {
	t.Helper()

	if gotErr == nil {
		t.Fatalf("%s %s : want error to be %v, got nil", testName, newPath, wantErr)
	}

	err, ok := gotErr.(*os.LinkError)
	if !ok {
		t.Fatalf("%s %s : want error type *os.LinkError,\n got %v", testName, newPath, reflect.TypeOf(gotErr))
	}

	if err.Op != wantOp || err.Err != wantErr {
		t.Errorf("%s %s %s : want error to be %v,\n got %v", testName, oldPath, newPath, wantErr, err)
	}
}

// CheckInvalid checks if the error in os.ErrInvalid.
func CheckInvalid(t *testing.T, funcName string, err error) {
	t.Helper()

	if err != os.ErrInvalid {
		t.Errorf("%s : want error to be %v, got %v", funcName, os.ErrInvalid, err)
	}
}

// CheckPanic checks that function f panics.
func CheckPanic(t *testing.T, funcName string, f func()) {
	t.Helper()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("%s : want function to panic, not panicing", funcName)
		}
	}()

	f()
}

// CreateEmptyFile creates an empty file and returns the file name.
func CreateEmptyFile(t *testing.T, vfs avfs.VFS, rootDir string) string {
	t.Helper()

	f, err := vfs.TempFile(rootDir, avfs.Avfs)
	if err != nil {
		t.Fatalf("TempFile : want error to be nil, got %v", err)
	}

	defer f.Close()

	return f.Name()
}

// CreateRndDir creates one directory with random empty subdirectories, files and symbolic links.
func CreateRndDir(t *testing.T, vfs avfs.VFS, rootDir string) *vfsutils.RndTree {
	t.Helper()

	RndParamsOneDir := vfsutils.RndTreeParams{
		MinDepth:    1,
		MaxDepth:    1,
		MinName:     4,
		MaxName:     32,
		MinDirs:     10,
		MaxDirs:     20,
		MinFiles:    50,
		MaxFiles:    100,
		MinFileLen:  0,
		MaxFileLen:  0,
		MinSymlinks: 5,
		MaxSymlinks: 10,
	}

	rndTree, err := vfsutils.NewRndTree(vfs, RndParamsOneDir)
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	err = rndTree.CreateTree(rootDir)
	if err != nil {
		t.Fatalf("rndTree.Create : want error to be nil, got %v", err)
	}

	return rndTree
}

func (sfs *SuiteFS) OpenNonExistingFile(t *testing.T) avfs.File {
	t.Helper()

	vfs := sfs.vfsRead
	name := vfs.Join(sfs.rootDir, "nonExistingFile")

	f, err := vfs.Open(name)
	if vfs.IsExist(err) {
		t.Fatalf("Open %s : want non existing file, got file exists", name)
	}

	return f
}
