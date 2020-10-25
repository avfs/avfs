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
	"testing"

	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
)

func TestMemFsWithNoIdm(t *testing.T) {
	vfs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	sidm := test.NewSuiteIdm(t, vfs)
	sidm.All()
}

func TestMemFsWithMemIdm(t *testing.T) {
	vfs, err := memfs.New(memfs.WithIdm(memidm.New()), memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	sidm := test.NewSuiteIdm(t, vfs)
	sidm.All()
}

func TestMemFsWithOsIdm(t *testing.T) {
	idm := osidm.New()
	if !idm.CurrentUser().IsRoot() {
		t.Skip("OsIdm only works on when connected as root on a linux platform, skipping")
	}

	vfs, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(idm))
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	sIdm := test.NewSuiteIdm(t, vfs)
	sIdm.All()
}
