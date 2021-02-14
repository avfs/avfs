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

package vfsutils_test

import (
	"testing"

	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfsutils"
)

func initTest(t *testing.T) *test.SuiteFS {
	vfs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(t, vfs)

	return sfs
}

func TestDirExists(t *testing.T) {
	sfs := initTest(t)

	rootDir, removeDir := sfs.CreateRootDir(t, test.UsrTest)
	defer removeDir()

	vfs := sfs.VFSTest()
	existingFile := sfs.CreateEmptyFile(t)

	t.Run("DirExistsDir", func(t *testing.T) {
		ok, err := vfsutils.DirExists(vfs, rootDir)
		if err != nil {
			t.Errorf("DirExists : want error to be nil, got %v", err)
		}

		if !ok {
			t.Error("DirExists : want DirExists to be true, got false")
		}
	})

	t.Run("DirExistsFile", func(t *testing.T) {
		ok, err := vfsutils.DirExists(vfs, existingFile)
		if err != nil {
			t.Errorf("DirExists : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("DirExists : want DirExists to be false, got true")
		}
	})

	t.Run("DirExistsNotExists", func(t *testing.T) {
		nonExistingFile := sfs.NonExistingFile(t)

		ok, err := vfsutils.DirExists(vfs, nonExistingFile)
		if err != nil {
			t.Errorf("DirExists : want error to be nil, got %v", err)
		}

		if ok {
			t.Error("DirExists : want DirExists to be false, got true")
		}
	})
}
