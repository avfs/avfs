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
	"testing"

	"github.com/avfs/avfs"
)

// NotImplemented tests non implemented functions.
func (sfs *SuiteFS) NotImplemented(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.GetFsRead()

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		name := vfs.Name()
		if name != avfs.NotImplemented {
			t.Errorf("Name : want name to be %s, got %s", avfs.NotImplemented, name)
		}

		clean := vfs.Clean(rootDir)
		if clean != rootDir {
			t.Errorf("Clean : want Clean to be %s, got %s", rootDir, clean)
		}

		dir := vfs.Dir(rootDir)
		if dir != string(avfs.PathSeparator) {
			t.Errorf("Dir : want Dir to be %s, got %s", string(avfs.PathSeparator), dir)
		}

		b := vfs.IsAbs(rootDir)
		if !b {
			t.Errorf("IsAbs : want result to be true, got false")
		}

		b = vfs.IsPathSeparator(avfs.PathSeparator)
		if !b {
			t.Errorf("IsPathSeparator : want result to be true, got false")
		}

		join := vfs.Join(rootDir, rootDir)
		if join != rootDir+rootDir {
			t.Errorf("Join : want join to be %s, got %s", rootDir+rootDir, join)
		}

		_, err := vfs.Rel(rootDir, rootDir)
		if err != nil {
			t.Errorf("Rel : want error to be nil, got %v", err)
		}

		dir, _ = vfs.Split(rootDir)
		if dir != string(avfs.PathSeparator) {
			t.Errorf("Split : want dir to be %c, got %s", avfs.PathSeparator, dir)
		}
	}

	if !vfs.HasFeature(avfs.FeatBasicFs) {
		f, _ := vfs.Open(rootDir)

		n := f.Name()
		if n != avfs.NotImplemented {
			t.Errorf("Name : want error to be %v, got %v", avfs.NotImplemented, n)
		}

		_, err := f.Stat()
		CheckPathError(t, "Stat", "stat", f.Name(), avfs.ErrPermDenied, err)

		err = f.Sync()
		CheckPathError(t, "Sync", "sync", f.Name(), avfs.ErrPermDenied, err)

		_, err = f.WriteString("")
		CheckPathError(t, "WriteString", "write", f.Name(), avfs.ErrPermDenied, err)
	}
}
