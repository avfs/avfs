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
//
// The tree is always built as the administrator of idm, because creating the
// system and user directories requires privileges; the returned file system
// then acts as Options.User.
func NewWithOptions(opts *Options) *MemFS {
	if opts == nil {
		opts = &Options{}
	}

	idm := opts.Idm
	if idm == nil {
		idm = memidm.New()
	}

	features := avfs.FeatHardlink | avfs.FeatSubFS | avfs.FeatSymlink | idm.Features() | avfs.BuildFeatures()

	admin := idm.AdminUser()

	storage := &Storage{name: opts.Name}
	vfs := &MemFS{storage: storage}

	// The identity of the file system is built first: the features and the
	// umask are set on it, and the tree is created by the administrator.
	userDir := &avfs.UserDirMixin{}

	// A foreign OS type is silently ignored in a build without the
	// avfs_setostype tag: the file system then emulates the host OS.
	_ = userDir.Init(opts.OSType, idm, admin)
	vfs.userDir = userDir

	_ = vfs.SetFeatures(features)
	_ = vfs.SetUMask(avfs.UMask())

	// The identity of the administrator is the one of the file system itself,
	// so that cloning it back does not build a second view of it.
	storage.userDirs = map[userDirKey]*avfs.UserDirMixin{
		{name: admin.Name(), uid: admin.Uid(), ost: userDir.OSType()}: userDir,
	}

	vfs.err = avfs.ErrorsFor(vfs.OSType())
	vfs.storage.rootNode = vfs.createRootNode()

	if vfs.OSType() == avfs.OsWindows {
		vfs.storage.volumes = make(volumes)
		vfs.storage.volumes[avfs.DefaultVolume] = vfs.storage.rootNode
	}

	systemDirs := opts.SystemDirs
	if len(systemDirs) == 0 {
		systemDirs = avfs.SystemDirs(vfs)
	}

	err := avfs.MkDirs(vfs, systemDirs, "")
	if err != nil {
		panic(err)
	}

	u := opts.User
	if u == nil {
		u = admin
	}

	// Create the user directories while still the administrator, so that the
	// Chown done by MkDirs is allowed, then hand the tree over to the target
	// user. A file system cannot change user, so this is a clone.
	err = avfs.MkDirs(vfs, avfs.UserDirs(vfs, u), "")
	if err != nil {
		panic(err)
	}

	if u.Name() == admin.Name() {
		return vfs
	}

	userVfs, err := vfs.cloneWithUser(u, vfs.OSType())
	if err != nil {
		panic(err)
	}

	return userVfs
}

// Name returns the name of the fileSystem.
func (vfs *MemFS) Name() string {
	if vfs.storage == nil {
		return ""
	}

	return vfs.storage.name
}

// Type returns the type of the fileSystem or Identity manager.
func (*MemFS) Type() string {
	return "MemFS"
}

// VolumeAdd adds a new volume to a Windows file system.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) VolumeAdd(name string) error {
	const op = "VolumeAdd"

	if vfs.OSType() != avfs.OsWindows {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrVolumeWindows}
	}

	vol := vfs.VolumeName(name)
	if vol == "" {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrVolumeNameInvalid}
	}

	vfs.storage.volMu.Lock()
	defer vfs.storage.volMu.Unlock()

	_, ok := vfs.storage.volumes[vol]
	if ok {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrVolumeAlreadyExists}
	}

	vfs.storage.volumes[vol] = vfs.createRootNode()

	return nil
}

// VolumeDelete deletes an existing volume and all its files from a Windows file system.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) VolumeDelete(name string) error {
	const op = "VolumeDelete"

	if vfs.OSType() != avfs.OsWindows {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrVolumeWindows}
	}

	vol := vfs.VolumeName(name)
	if vol == "" {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrVolumeNameInvalid}
	}

	vfs.storage.volMu.RLock()
	_, ok := vfs.storage.volumes[vol]
	vfs.storage.volMu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrVolumeNameInvalid}
	}

	err := vfs.RemoveAll(vol)
	if err != nil {
		return err
	}

	vfs.storage.volMu.Lock()
	delete(vfs.storage.volumes, vol)
	vfs.storage.volMu.Unlock()

	return nil
}

// VolumeList returns the volumes of the file system.
func (vfs *MemFS) VolumeList() []string {
	var l []string

	if vfs.OSType() != avfs.OsWindows {
		return l
	}

	vfs.storage.volMu.RLock()
	defer vfs.storage.volMu.RUnlock()

	for v := range vfs.storage.volumes {
		l = append(l, v)
	}

	return l
}
