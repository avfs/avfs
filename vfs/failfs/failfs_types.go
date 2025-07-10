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
	"io/fs"
	"time"

	"github.com/avfs/avfs"
)

// FailFS implements a failing file system using the avfs.VFS interface.
// It fails if the FailFunc function returns an error.
type FailFS struct {
	baseFS          avfs.VFS // baseFS is the base file system.
	failFunc        FailFunc // failFunc is the function
	avfs.FeaturesFn          // FeaturesFn provides features functions to a file system or an identity manager.
}

// FailFile represents an open file descriptor.
type FailFile struct {
	baseFile avfs.File // baseFile represents an open file descriptor from the base file system.
	vfs      *FailFS   // vfs is the base path file system of the file.
}

// FailFunc is a type of function that returns an error depending on the parameters of the caller function.
// If this function returns nil the corresponding function of the base file system is called.
// If an error is returned, the caller function return this error without executing the base function.
type FailFunc func(vfs avfs.VFSBase, fn avfs.FnVFS, failParam *FailParam) error

// FailParam regroups all possible parameters passed to the functions of a file system.
type FailParam struct {
	ATime   time.Time   // ATime is used in the Chtimes functions.
	MTime   time.Time   // MTime is used in the Chtimes functions.
	Op      string      // Op is the operation used for fs.PathError or os.LinkError.
	Path    string      // Path is the Path parameter (or old path for link functions).
	NewPath string      // NewPath is the New parameter for link functions.
	Flag    int         // Flag is the opening flag for Open function.
	Uid     int         // Uid is used in the Chown and Lchown functions.
	Gid     int         // Gid is used in the Chown and Lchown functions.
	Size    int64       // Size is used in Truncate functions.
	Perm    fs.FileMode // Perm represents the permission mode parameter.
}
