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

package basepathfs

import (
	"github.com/avfs/avfs"
)

// BasePathFS implements a base path file system.
type BasePathFS struct {
	baseFS   avfs.VFS
	basePath string
	feature  avfs.Feature
}

// BasePathFile represents an open file descriptor.
type BasePathFile struct {
	bpFS     *BasePathFS
	baseFile avfs.File
}
