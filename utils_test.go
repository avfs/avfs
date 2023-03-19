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

//go:build !datarace

package avfs_test

import (
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/orefafs"
)

func TestUtilsMemFS(t *testing.T) {
	vfs := memfs.New()

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestAll(t)
}

func TestUtilsOrefaFS(t *testing.T) {
	vfs := orefafs.New()

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestAll(t)
}

// TestUMaskOS tests Umask functions for the current OS.
func TestUMaskOS(t *testing.T) {
	const (
		defaultUmask = fs.FileMode(0o22) // defaultUmask is the default umask.
		umaskSet     = fs.FileMode(0o77)
	)

	umaskTest := umaskSet

	if avfs.CurrentOSType() == avfs.OsWindows {
		umaskTest = defaultUmask
	}

	umask := avfs.UMask()
	if umask != defaultUmask {
		t.Errorf("UMask : want OS umask %o, got %o", defaultUmask, umask)
	}

	avfs.SetUMask(umaskSet)

	umask = avfs.UMask()
	if umask != umaskTest {
		t.Errorf("UMask : want test umask %o, got %o", umaskTest, umask)
	}

	avfs.SetUMask(defaultUmask)

	umask = avfs.UMask()
	if umask != defaultUmask {
		t.Errorf("UMask : want OS umask %o, got %o", defaultUmask, umask)
	}
}
