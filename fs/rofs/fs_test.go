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

package rofs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/fs/rofs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
)

var (
	// rofs.RoFs struct implements avfs.Fs interface
	_ avfs.Fs = &rofs.RoFs{}

	// rofs.RoFile struct implements avfs.File interface
	_ avfs.File = &rofs.RoFile{}
)

func initTest(t *testing.T) *test.ConfigFs {
	fsRoot, err := memfs.New(memfs.OptIdm(memidm.New()), memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("New : want err to be nil, got %v", err)
	}

	cf := test.NewConfigFs(t, fsRoot)
	fsW := cf.GetFsWrite()
	fsR := rofs.New(fsW)
	cf.FsRead(fsR)

	return cf
}

func TestRoFs(t *testing.T) {
	cf := initTest(t)
	cf.SuiteRead()
	cf.SuiteReadOnly()
	cf.SuiteEvalSymlink()
	cf.SuitePath()
}

func TestRoFsPerm(t *testing.T) {
	cf := initTest(t)
	cf.SuitePermRead()
}
