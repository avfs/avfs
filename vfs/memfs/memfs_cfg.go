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
func New() (*MemFS, error) {
	return NewWithOptions(nil)
}

// NewWithOptions returns a new memory file system (MemFS) with the selected Options.
//
// The tree is always built as the administrator of idm, because creating the
// system and user directories requires privileges; the returned file system
// then acts as Options.User.
func NewWithOptions(opts *Options) (*MemFS, error) {
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
		return nil, err
	}

	u := opts.User
	if u == nil {
		u = admin
	}

	// Create the user directories while still the administrator, so that the
	// Chown done by MkDirs is allowed, then hand the tree over to the target
	// user. A file system cannot change user, so this is a clone.
	err = avfs.MkDirs(vfs, avfs.UserDirsInfo(vfs, u), "")
	if err != nil {
		return nil, err
	}

	if u.Name() == admin.Name() {
		return vfs, nil
	}

	userVfs, err := vfs.cloneWithUser(u, vfs.OSType())
	if err != nil {
		return nil, err
	}

	return userVfs, nil
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
// If there is an error, it will be of type [*PathError].
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
// If there is an error, it will be of type [*PathError].
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

// CloneWithUser returns a shallow copy of the current file system (see MemFs)
// acting as user and emulating ost.
//
// The clone shares the Storage (the nodes, the volumes and the identity views)
// of the file system it is copied from, so a file created through either of them
// is visible from both. Cloning twice for the same user and OS type returns two
// file systems sharing the same identity view, hence the same current
// directory: Chdir on one of them is visible in the other.
//
// The clone starts in the home directory of user, which is created in the
// shared content if it does not exist yet.
//
// If user is nil, the administrator of the identity manager is used. If ost is
// avfs.OsUnknown, the OS type of the current file system is kept. If ost can't
// be set (see avfs.ErrSetOSType), it returns that error.
//
// The content is not converted to the new OS type: a view emulating another OS
// resolves paths with the rules, and reports the errors, of that OS, but the
// directories already created keep the names and the layout of the OS the
// content was built for.
func (vfs *MemFS) CloneWithUser(user avfs.UserReader, ost avfs.OSType) (avfs.VFS, error) {
	return vfs.cloneWithUser(user, ost)
}

// CloneWithUserName returns a shallow copy of the current file system (see
// MemFs) acting as the user userName and emulating ost.
//
// If the user is not found, the returned error is of type avfs.UnknownUserError,
// or avfs.ErrPermDenied if the identity manager of the file system holds no
// user.
func (vfs *MemFS) CloneWithUserName(userName string, ost avfs.OSType) (avfs.VFS, error) {
	// A file system whose identity manager holds no user answers
	// ErrPermDenied, as LookupUser does.
	user, err := vfs.Idm().LookupUser(userName)
	if err != nil {
		return nil, err
	}

	return vfs.cloneWithUser(user, ost)
}

// cloneWithUser returns a view of the file system acting as user and emulating
// ost, sharing its Storage.
func (vfs *MemFS) cloneWithUser(user avfs.UserReader, ost avfs.OSType) (*MemFS, error) {
	if ost == avfs.OsUnknown {
		ost = vfs.OSType()
	}

	if user == nil {
		user = vfs.Idm().AdminUser()
	}

	userDir, err := vfs.storage.userDirFor(vfs.Idm(), user, ost, vfs.Features(), vfs.UMask())
	if err != nil {
		return nil, err
	}

	clone := &MemFS{
		err:     avfs.ErrorsFor(userDir.OSType()),
		storage: vfs.storage,
		userDir: userDir,
	}

	err = vfs.createUserDirs(user, userDir.OSType())
	if err != nil {
		return nil, err
	}

	return clone, nil
}

// createUserDirs creates the directories of user in the shared content if they
// do not exist yet: its home directory, which a clone starts in, and the
// user-specific directories of the OS, such as the temporary directory on
// Windows and Darwin.
//
// The directory is created by the administrator of the identity manager, through
// a view of the storage emulating ost: a view acting as user may have neither
// the privileges to create it in its parent directory nor the right to own it.
//
// Nothing is created when ost is not the OS type of the file system it is
// cloned from: the content is not converted to another OS (see the AVFS
// specification), so the directories of that OS are not created either.
func (vfs *MemFS) createUserDirs(user avfs.UserReader, ost avfs.OSType) error {
	if ost != vfs.OSType() {
		return nil
	}

	admin := vfs.Idm().AdminUser()

	userDir, err := vfs.storage.userDirFor(vfs.Idm(), admin, ost, vfs.Features(), vfs.UMask())
	if err != nil {
		return err
	}

	adminVfs := &MemFS{
		err:     avfs.ErrorsFor(ost),
		storage: vfs.storage,
		userDir: userDir,
	}

	return avfs.MkDirs(adminVfs, avfs.UserDirsInfo(adminVfs, user), "")
}
