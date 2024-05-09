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

package failfs

import (
	"github.com/avfs/avfs"
)

// New returns a new FailFS file system from a baseFS file system.
// The failure function is initially set to OkFunc and should be set by FailFS.SetFailFunc.
func New(baseFS avfs.VFS) *FailFS {
	vfs := &FailFS{
		baseFS:   baseFS,
		failFunc: OkFunc,
	}

	_ = vfs.SetFeatures(baseFS.Features())

	return vfs
}

// fail calls the FailFunc function set by SetFailFunc.
func (vfs *FailFS) fail(fn avfs.FnVFS, fp *FailParam) error {
	err := vfs.failFunc(vfs, fn, fp)

	return err
}

// Name returns the name of the fileSystem.
func (vfs *FailFS) Name() string {
	return vfs.baseFS.Name()
}

// Type returns the type of the fileSystem or Identity manager.
func (*FailFS) Type() string {
	return "FailFS"
}
