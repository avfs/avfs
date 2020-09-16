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

package orefafs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/orefafs"
	"github.com/avfs/avfs/test"
)

var (
	// orefafs.OrefaFs struct implements avfs.OrefaFs interface.
	_ avfs.Fs = &orefafs.OrefaFs{}

	// orefafs.MemFile struct implements avfs.File interface.
	_ avfs.File = &orefafs.OrefaFile{}
)

func initTest(t *testing.T) avfs.Fs {
	fs, err := orefafs.New(orefafs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	return fs
}

func TestOrefaFs(t *testing.T) {
	fs := initTest(t)

	sfs := test.NewSuiteFs(t, fs)
	sfs.All()
}

func TestNilPtrReceiver(t *testing.T) {
	f := (*orefafs.OrefaFile)(nil)

	test.SuiteNilPtrFile(t, f)
}

func TestOrefaFsFeatures(t *testing.T) {
	fs, err := orefafs.New()
	if err != nil {
		t.Fatalf("orefaFs.New : want error to be nil, got %v", err)
	}

	if fs.Features() != avfs.FeatBasicFs|avfs.FeatHardlink {
		t.Errorf("Features : want Features to be %d, got %d", avfs.FeatBasicFs, fs.Features())
	}
}

func TestOrefaFsOSType(t *testing.T) {
	fs, err := orefafs.New()
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	ost := fs.OSType()
	if ost != avfs.OsLinux {
		t.Errorf("OSType : want os type to be %v, got %v", avfs.OsLinux, ost)
	}
}

func BenchmarkOrefaFsCreate(b *testing.B) {
	fs, err := orefafs.New(orefafs.WithMainDirs())
	if err != nil {
		b.Fatalf("New : want error to be nil, got %v", err)
	}

	test.BenchmarkCreate(b, fs)
}
