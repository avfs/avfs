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

package avfs

import (
	"io/fs"
	"sync"
)

// UMaskType is the file mode creation mask.
// it must be set to be read, so it must be protected with a mutex.
type UMaskType struct {
	once sync.Once
	mu   sync.RWMutex
	mask fs.FileMode
}

// UMask is the global variable containing the file mode creation mask.
var UMask UMaskType //nolint:gochecknoglobals // Used by UMaskType Get and Set.

// Get returns the file mode creation mask.
func (um *UMaskType) Get() fs.FileMode {
	um.once.Do(func() {
		um.Set(0)
	})

	um.mu.RLock()
	u := um.mask
	um.mu.RUnlock()

	return u
}
