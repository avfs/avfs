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

package memfs

import (
	"io/fs"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
)

// New returns a new memory file system (MemFS) with the default Options.
func New() *MemFS {
	return NewWithOptions(nil)
}

// NewWithOptions returns a new memory file system (MemFS) with the selected Options.
func NewWithOptions(opts *Options) *MemFS {
	if opts == nil {
		opts = &Options{UMask: avfs.UMask(), OSType: avfs.OsUnknown}
	}

	idm := opts.Idm
	if idm == nil {
		idm = memidm.New()
	}

	features := avfs.FeatHardlink | avfs.FeatSubFS | avfs.FeatSymlink | idm.Features() | avfs.BuildFeatures()

	user := opts.User
	if opts.User == nil {
		user = idm.AdminUser()
	}

	vfs := &MemFS{
		dirMode:  fs.ModeDir,
		fileMode: 0,
		lastId:   new(uint64),
		name:     opts.Name,
	}

	_ = vfs.SetFeatures(features)
	_ = vfs.SetOSType(opts.OSType)
	_ = vfs.SetUMask(opts.UMask)
	_ = vfs.SetIdm(idm)
	_ = vfs.SetUser(user)

	vfs.err.SetOSType(vfs.OSType())
	vfs.rootNode = vfs.createRootNode()

	var volumeName string

	if vfs.OSType() == avfs.OsWindows {
		vfs.dirMode |= avfs.DefaultDirPerm
		vfs.fileMode |= avfs.DefaultFilePerm

		volumeName = avfs.DefaultVolume
		vfs.volumes = make(volumes)
		vfs.volumes[volumeName] = vfs.rootNode
	}

	_ = avfs.MkSystemDirs(vfs, opts.SystemDirs)

	return vfs
}

// Name returns the name of the fileSystem.
func (vfs *MemFS) Name() string {
	return vfs.name
}

// Type returns the type of the fileSystem or Identity manager.
func (*MemFS) Type() string {
	return "MemFS"
}

// VolumeAdd adds a new volume to a Windows file system.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) VolumeAdd(path string) error {
	const op = "VolumeAdd"

	if vfs.OSType() != avfs.OsWindows {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrVolumeWindows}
	}

	vol := avfs.VolumeName(vfs, path)
	if vol == "" {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrVolumeNameInvalid}
	}

	_, ok := vfs.volumes[vol]
	if ok {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrVolumeAlreadyExists}
	}

	vfs.volumes[vol] = vfs.createRootNode()

	return nil
}

// VolumeDelete deletes an existing volume and all its files from a Windows file system.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) VolumeDelete(path string) error {
	const op = "VolumeDelete"

	if vfs.OSType() != avfs.OsWindows {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrVolumeWindows}
	}

	vol := avfs.VolumeName(vfs, path)
	if vol == "" {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrVolumeNameInvalid}
	}

	_, ok := vfs.volumes[vol]
	if !ok {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrVolumeNameInvalid}
	}

	err := vfs.RemoveAll(vol)
	if err != nil {
		return err
	}

	delete(vfs.volumes, vol)

	return nil
}

// VolumeList returns the volumes of the file system.
func (vfs *MemFS) VolumeList() []string {
	var l []string //nolint:prealloc // Consider preallocating `l`

	if vfs.OSType() != avfs.OsWindows {
		return l
	}

	for v := range vfs.volumes {
		l = append(l, v)
	}

	return l
}
