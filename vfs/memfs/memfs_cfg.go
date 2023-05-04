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
		opts = &Options{SystemDirs: true}
	}

	features := avfs.FeatHardlink | avfs.FeatSubFS | avfs.FeatSymlink | avfs.BuildFeatures()
	if opts.SystemDirs {
		features |= avfs.FeatSystemDirs
	}

	idm := opts.Idm
	if idm == nil {
		idm = memidm.New()
	}

	features |= idm.Features()

	user := opts.User
	if opts.User == nil {
		user = idm.AdminUser()
	}

	ma := &memAttrs{
		idm:      idm,
		dirMode:  fs.ModeDir,
		fileMode: 0,
		name:     opts.Name,
	}

	vfs := &MemFS{
		memAttrs: ma,
		curDir:   "/",
		umask:    avfs.UMask(),
		user:     user,
	}

	vfs.SetFeatures(features)
	vfs.SetOSType(opts.OSType)
	vfs.err.SetOSType(vfs.OSType())
	vfs.rootNode = vfs.createRootNode()

	volumeName := ""

	if vfs.OSType() == avfs.OsWindows {
		ma.dirMode |= avfs.DefaultDirPerm
		ma.fileMode |= avfs.DefaultFilePerm

		volumeName = avfs.DefaultVolume
		vfs.curDir = volumeName + string(vfs.PathSeparator())
		vfs.volumes = make(volumes)
		vfs.volumes[volumeName] = vfs.rootNode
	}

	if vfs.HasFeature(avfs.FeatSystemDirs) {
		// Save the current user and umask.
		u := vfs.user
		um := vfs.umask

		// Create system directories as administrator user without umask.
		vfs.user = ma.idm.AdminUser()
		vfs.umask = 0

		err := vfs.CreateSystemDirs(volumeName)
		if err != nil {
			panic("CreateSystemDirs " + err.Error())
		}

		// Restore the previous user and umask.
		vfs.umask = um
		vfs.user = u
		vfs.curDir = vfs.Utils.HomeDirUser(u)
	}

	return vfs
}

// Name returns the name of the fileSystem.
func (vfs *MemFS) Name() string {
	return vfs.memAttrs.name
}

// Type returns the type of the fileSystem or Identity manager.
func (*MemFS) Type() string {
	return "MemFS"
}

// Configuration functions.

// CreateSystemDirs creates the system directories of a file system.
func (vfs *MemFS) CreateSystemDirs(basePath string) error {
	return vfs.Utils.CreateSystemDirs(vfs, basePath)
}

// CreateHomeDir creates and returns the home directory of a user.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) CreateHomeDir(u avfs.UserReader) (string, error) {
	return vfs.Utils.CreateHomeDir(vfs, u)
}

// VolumeAdd adds a new volume to a Windows file system.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) VolumeAdd(path string) error {
	const op = "VolumeAdd"

	if vfs.OSType() != avfs.OsWindows {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrVolumeWindows}
	}

	vol := vfs.Utils.VolumeName(path)
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

	vol := vfs.VolumeName(path)
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
