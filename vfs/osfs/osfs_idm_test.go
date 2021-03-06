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

//go:build !datarace
// +build !datarace

package osfs_test

import (
	"testing"

	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/osfs"
)

// TestOsFsWithOsIdm tests OsFS identity manager functions with OsIdn identity manager.
func TestIdmOsFSWithOsIdm(t *testing.T) {
	vfs, err := osfs.New(osfs.WithIdm(osidm.New()))
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	sidm := test.NewSuiteIdm(t, vfs)
	sidm.TestAll(t)
}

// TestOsFSWithoutIdm test OsFS without and identity manager.
func TestIdmOsFSWithNoIdm(t *testing.T) {
	vfs, err := osfs.New()
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	sidm := test.NewSuiteIdm(t, vfs)
	sidm.TestAll(t)
}
