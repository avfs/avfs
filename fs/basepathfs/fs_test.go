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

package basepathfs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/basepathfs"
	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
)

var (
	// basepathfs.BasePathFs struct implements avfs.MemFs interface.
	_ avfs.Fs = &basepathfs.BasePathFs{}

	// basepathfs.BasePathFile struct implements avfs.File interface.
	_ avfs.File = &basepathfs.BasePathFile{}
)

// initTest
func initTest(t *testing.T) *basepathfs.BasePathFs {
	const basePath = "/base/path"

	baseFs, err := memfs.New(memfs.OptIdm(memidm.New()), memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	err = baseFs.MkdirAll(basePath, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("MkdirAll %s : want error to be nil, got %v", basePath, err)
	}

	fs, err := basepathfs.New(baseFs, basePath)
	if err != nil {
		t.Fatalf("basepathfs.New : want error to be nil, got %v", err)
	}

	return fs
}

// TestMemFs
func TestBasePathFs(t *testing.T) {
	fs := initTest(t)
	cf := test.NewConfigFs(t, fs)

	cf.SuiteAll()
}

// TestMemFsPerm
func TestBasePathFsPerm(t *testing.T) {
	fs := initTest(t)
	cf := test.NewConfigFs(t, fs)

	cf.SuitePerm()
}

func TestBasePathFsConfig(t *testing.T) {
	const (
		nonExistingDir = "/non/existing/dir"
		existingFile   = "/tmp/existingFile"
		basePathTmp    = "/tmp/base/path/tmp"
	)

	fs, err := memfs.New(memfs.OptIdm(memidm.New()), memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	err = fs.WriteFile(existingFile, []byte{}, avfs.DefaultFilePerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	_, err = basepathfs.New(fs, nonExistingDir)
	test.CheckPathError(t, "BasePath", "basepath", nonExistingDir, avfs.ErrNoSuchFileOrDir, err)

	_, err = basepathfs.New(fs, existingFile)
	test.CheckPathError(t, "BasePath", "basepath", existingFile, avfs.ErrNotADirectory, err)

	err = fs.MkdirAll(basePathTmp, avfs.DefaultDirPerm)
	if err != nil {
		t.Fatalf("WriteFile : want error to be nil, got %v", err)
	}

	_, err = basepathfs.New(fs, fs.Dir(basePathTmp))
	test.CheckPathError(t, "BasePath", "mkdir", "/tmp", avfs.ErrFileExists, err)
}
