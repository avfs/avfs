//
//  Copyright 2022 The AVFS authors
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

package mountfs

import (
	"sync"

	"github.com/avfs/avfs"
)

// MountFS implements a memory file system using the avfs.VFS interface.
type MountFS struct {
	rootFS  avfs.VFS
	mounts  mounts
	rootMnt *mount
	curMnt  *mount
	name    string
	mu      sync.RWMutex
	avfs.VFSFn[*MountFS]
}

// mounts are the mount points of the MountFS file system.
type mounts map[string]*mount

// mount is the mount point of a file system.
type mount struct {
	vfs      avfs.VFS
	mntPath  string
	basePath string
}

// MountFile represents an open file descriptor.
type MountFile struct {
	vfs   *MountFS
	mount *mount
	file  avfs.File
}
