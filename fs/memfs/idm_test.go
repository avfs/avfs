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

// +build !datarace

package memfs_test

import (
	"runtime"
	"testing"

	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
)

func TestMemFsWithNoIdm(t *testing.T) {
	fs, err := memfs.New(memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	ci := test.NewConfigIdm(t, fs)
	ci.SuiteAll()
}

func TestMemFsWithMemIdm(t *testing.T) {
	fs, err := memfs.New(memfs.OptIdm(memidm.New()), memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	ci := test.NewConfigIdm(t, fs)
	ci.SuiteAll()
}

func TestMemFsWithOsIdm(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("OsIdm only works on a linux platform, skipping")
	}

	fs, err := memfs.New(memfs.OptIdm(osidm.New()), memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	ci := test.NewConfigIdm(t, fs)
	ci.SuiteAll()
}
