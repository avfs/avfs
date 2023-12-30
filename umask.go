//
//  Copyright 2023 The AVFS authors
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

package avfs

import (
	"io/fs"
	"sync/atomic"
)

// UMasker is the interface that wraps umask related methods.
type UMasker interface {
	// SetUMask sets the file mode creation mask.
	SetUMask(mask fs.FileMode) error

	// UMask returns the file mode creation mask.
	UMask() fs.FileMode
}

// UMaskFn provides UMask functions to file systems.
type UMaskFn struct {
	umask fs.FileMode // umask is the user file creation mode mask.
}

// SetUMask sets the file mode creation mask.
func (umf *UMaskFn) SetUMask(mask fs.FileMode) error {
	atomic.StoreUint32((*uint32)(&umf.umask), uint32(mask))

	return nil
}

// UMask returns the file mode creation mask.
func (umf *UMaskFn) UMask() fs.FileMode {
	m := atomic.LoadUint32((*uint32)(&umf.umask))

	return fs.FileMode(m)
}
