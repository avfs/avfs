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

//go:build !go1.23

package avfs

import _ "unsafe" // for go:linkname only.

// nextRandom is used in avfs.CreateTemp and avfs.MkdirTemp.
//
//go:linkname nextRandom os.nextRandom
func nextRandom() string

//go:linkname volumeNameLen path/filepath.volumeNameLen
func volumeNameLen(path string) int
