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

package orefafs

import (
	"io/fs"

	"github.com/avfs/avfs"
)

// This file forwards the methods of avfs.UserDirMixin to OrefaFS.
//
// OrefaFS holds its identity in a userDir field rather than embedding the
// mixin, so that the state which is private to a file system (its user, home
// and temporary directories, its current directory, its emulated OS type and
// its identity manager) is explicit, and so that a clone is obviously a new
// file system rather than a copy of a struct holding a lock.
//
// The methods below are the interface avfs.VFSBase requires from that mixin.
// They must be kept in sync with it.

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path.
// Abs calls Clean on the result.
func (vfs *OrefaFS) Abs(path string) (string, error) {
	return vfs.userDir.Abs(path)
}

// Base returns the last element of path.
func (vfs *OrefaFS) Base(path string) string {
	return vfs.userDir.Base(path)
}

// Clean returns the shortest path name equivalent to path.
func (vfs *OrefaFS) Clean(path string) string {
	return vfs.userDir.Clean(path)
}

// CurDir returns the current directory.
func (vfs *OrefaFS) CurDir() string {
	return vfs.userDir.CurDir()
}

// Dir returns all but the last element of path.
func (vfs *OrefaFS) Dir(path string) string {
	return vfs.userDir.Dir(path)
}

// DirMode returns the default file mode for new directories.
func (vfs *OrefaFS) DirMode() fs.FileMode {
	return vfs.userDir.DirMode()
}

// Features returns the set of features provided by the file system.
func (vfs *OrefaFS) Features() avfs.Features {
	return vfs.userDir.Features()
}

// FileMode returns the default file mode for new files.
func (vfs *OrefaFS) FileMode() fs.FileMode {
	return vfs.userDir.FileMode()
}

// FromSlash returns the result of replacing each slash ('/') character in path
// with a separator character.
func (vfs *OrefaFS) FromSlash(path string) string {
	return vfs.userDir.FromSlash(path)
}

// Getwd returns an absolute path name corresponding to the current directory.
func (vfs *OrefaFS) Getwd() (string, error) {
	return vfs.userDir.Getwd()
}

// HasFeature returns true if the file system provides a given feature.
func (vfs *OrefaFS) HasFeature(feature avfs.Features) bool {
	return vfs.userDir.HasFeature(feature)
}

// Idm returns the identity manager of the file system.
func (vfs *OrefaFS) Idm() avfs.IdmMgr {
	return vfs.userDir.Idm()
}

// InitIdm sets the identity manager of the file system.
// It must be called once, during construction.
func (vfs *OrefaFS) InitIdm(idm avfs.IdmMgr) error {
	return vfs.userDir.InitIdm(idm)
}

// InitOSType sets the OS type of the file system.
// It must be called once, during construction.
func (vfs *OrefaFS) InitOSType(ost avfs.OSType) error {
	return vfs.userDir.InitOSType(ost)
}

// IsAbs reports whether the path is absolute.
func (vfs *OrefaFS) IsAbs(path string) bool {
	return vfs.userDir.IsAbs(path)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *OrefaFS) IsPathSeparator(c uint8) bool {
	return vfs.userDir.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path.
func (vfs *OrefaFS) Join(elem ...string) string {
	return vfs.userDir.Join(elem...)
}

// Match reports whether name matches the shell file name pattern.
func (vfs *OrefaFS) Match(pattern, name string) (bool, error) {
	return vfs.userDir.Match(pattern, name)
}

// OSType returns the operating system type of the file system.
func (vfs *OrefaFS) OSType() avfs.OSType {
	return vfs.userDir.OSType()
}

// PathSeparator returns the OS-specific path separator.
func (vfs *OrefaFS) PathSeparator() uint8 {
	return vfs.userDir.PathSeparator()
}

// Rel returns a relative path in lexical form.
func (vfs *OrefaFS) Rel(basepath, targpath string) (string, error) {
	return vfs.userDir.Rel(basepath, targpath)
}

// SetCurDir sets the current directory.
//
// Unlike Chdir, it resolves nothing and checks no permission: the caller is
// responsible for having authorized the directory beforehand.
func (vfs *OrefaFS) SetCurDir(curDir string) error {
	return vfs.userDir.SetCurDir(curDir)
}

// SetFeatures sets the features of the file system.
func (vfs *OrefaFS) SetFeatures(feature avfs.Features) error {
	return vfs.userDir.SetFeatures(feature)
}

// SetUMask sets the file mode creation mask.
func (vfs *OrefaFS) SetUMask(mask fs.FileMode) error {
	return vfs.userDir.SetUMask(mask)
}

// Split splits path immediately following the final Separator.
func (vfs *OrefaFS) Split(path string) (dir, file string) {
	return vfs.userDir.Split(path)
}

// TempDir returns the default directory to use for temporary files.
func (vfs *OrefaFS) TempDir() string {
	return vfs.userDir.TempDir()
}

// ToSlash returns the result of replacing each separator character in path
// with a slash ('/') character.
func (vfs *OrefaFS) ToSlash(path string) string {
	return vfs.userDir.ToSlash(path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that
// implements interface avfs.SysStater.
func (vfs *OrefaFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return vfs.userDir.ToSysStat(info)
}

// UMask returns the file mode creation mask.
func (vfs *OrefaFS) UMask() fs.FileMode {
	return vfs.userDir.UMask()
}

// User returns the current user.
func (vfs *OrefaFS) User() avfs.UserReader {
	return vfs.userDir.User()
}

// UserHomeDir returns the current user's home directory.
func (vfs *OrefaFS) UserHomeDir() (string, error) {
	return vfs.userDir.UserHomeDir()
}

// VolumeName returns the leading volume name.
func (vfs *OrefaFS) VolumeName(path string) string {
	return vfs.userDir.VolumeName(path)
}

// VolumeNameLen returns the length of the leading volume name on Windows.
func (vfs *OrefaFS) VolumeNameLen(path string) int {
	return vfs.userDir.VolumeNameLen(path)
}
