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

package memfs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/idm/dummyidm"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
)

var (
	// memfs.MemFs struct implements avfs.MemFs interface.
	_ avfs.Fs = &memfs.MemFs{}

	// memfs.MemFile struct implements avfs.File interface.
	_ avfs.File = &memfs.MemFile{}
)

// initTest
func initTest(t *testing.T) *test.ConfigFs {
	fsRoot, err := memfs.New(memfs.OptIdm(memidm.New()), memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	cf := test.NewConfigFs(t, fsRoot)

	return cf
}

// TestMemFs
func TestMemFs(t *testing.T) {
	cf := initTest(t)
	cf.SuiteAll()
}

// TestMemFsPerm
func TestMemFsPerm(t *testing.T) {
	cf := initTest(t)
	cf.SuitePerm()
}

// TestMemFsOptionError
func TestMemFsOptionError(t *testing.T) {
	_, err := memfs.New(memfs.OptIdm(dummyidm.New()))
	if err != avfs.ErrPermDenied {
		t.Errorf("New : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}
}

// TestMemFsOptionName
func TestMemFsOptionName(t *testing.T) {
	const wantName = "whatever"

	fs, err := memfs.New()
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	if fs.Name() != "" {
		t.Errorf("New : want name to be '', got %s", fs.Name())
	}

	fs, err = memfs.New(memfs.OptName(wantName))
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	name := fs.Name()
	if name != wantName {
		t.Errorf("New : want name to be %s, got %s", wantName, fs.Name())
	}
}

// TestNilPtrReceiver
func TestNilPtrReceiver(t *testing.T) {
	f := (*memfs.MemFile)(nil)

	test.SuiteNilPtrFile(t, f)
}

// TestMemFsFeatures
func TestMemFsFeatures(t *testing.T) {
	fs, err := memfs.New()
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	if fs.Features()&avfs.FeatIdentityMgr != 0 {
		t.Errorf("Features : want FeatIdentityMgr missing, got present")
	}

	fs, err = memfs.New(memfs.OptIdm(memidm.New()))
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	if fs.Features()&avfs.FeatIdentityMgr == 0 {
		t.Errorf("Features : want FeatIdentityMgr present, got missing")
	}
}

// TestRace
func TestRace(t *testing.T) {
	cf := initTest(t)

	cf.SuiteRace()
}

// BenchmarkMemFsCreate
func BenchmarkMemFsCreate(b *testing.B) {
	fs, err := memfs.New(memfs.OptMainDirs())
	if err != nil {
		b.Fatalf("New : want error to be nil, got %v", err)
	}

	test.BenchmarkCreate(b, fs)
}
