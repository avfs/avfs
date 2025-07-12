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
	err             *avfs.ErrorsForOS // err regroups errors depending on the OS emulated.
	baseFS          avfs.VFS          // baseFS is the base file system.
	basePath        string            // basePath is the absolute path prepended to all files of the base file system.
	avfs.FeaturesFn                   // FeaturesFn provides features functions to a file system or an identity manager.
}

// BasePathFile represents an open file descriptor.
type BasePathFile struct {
	baseFile avfs.File   // baseFile represents an open file descriptor from the base file system.
	vfs      *BasePathFS // vfs is the base path file system of the file.
}
