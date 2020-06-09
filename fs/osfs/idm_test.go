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

package osfs_test

import (
	"testing"

	"github.com/avfs/avfs/fs/osfs"
	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
)

// TestOsFsWithOsIdm tests OsFs identity manager functions with OsIdn identity manager.
func TestOsFsWithOsIdm(t *testing.T) {
	fs, err := osfs.New(osfs.OptIdm(osidm.New()))
	if err != nil {
		t.Fatalf("New : want err to be nil, got %s", err)
	}

	ci := test.NewConfigIdm(t, fs)
	ci.SuiteAll()
}

// TestOsFsWithoutIdm
func TestOsFsWithoutIdm(t *testing.T) {
	fs, err := osfs.New()
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	ci := test.NewConfigIdm(t, fs)
	ci.SuitePermDenied()
}
