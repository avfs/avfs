//
//  Copyright 2024 The AVFS authors
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

//go:build !avfs_race

package failfs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/failfs"
	"github.com/avfs/avfs/vfs/orefafs"
)

var (
	// Tests that failfs.FailFS struct implements avfs.VFS interface.
	_ avfs.VFS = &failfs.FailFS{}

	// Tests that failfs.FailFS struct implements avfs.VFSBase interface.
	_ avfs.VFSBase = &failfs.FailFS{}

	// Tests that failfs.FailFile struct implements avfs.File interface.
	_ avfs.File = &failfs.FailFile{}
)

func TestFailFSNoFail(t *testing.T) {
	baseFS := orefafs.New()
	vfs := failfs.New(baseFS)

	ts := test.NewSuiteFS(t, vfs, vfs)
	ts.TestVFSAll(t)
}

func TestFailFSReadOnly(t *testing.T) {
	baseFS := orefafs.New()
	vfs := failfs.New(baseFS)

	features := baseFS.Features()
	_ = vfs.SetFeatures(features&^avfs.FeatIdentityMgr | avfs.FeatReadOnly)

	_ = vfs.SetFailFunc(failfs.ReadOnlyFunc)

	ts := test.NewSuiteFS(t, baseFS, vfs)
	ts.TestVFSAll(t)
}
