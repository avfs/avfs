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

// Package mountfs implements a file system composed of different file systems.
package mountfs

import (
	"io/fs"
	"os"
	"time"

	"github.com/avfs/avfs"
)

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Chdir(dir string) error {
	mnt, vfsPath := vfs.pathToMount(dir)

	err := mnt.vfs.Chdir(vfsPath)
	if err != nil {
		return mnt.restoreError(err)
	}

	vfs.curMnt = mnt
	_ = vfs.SetCurDir(vfsPath)

	return nil
}

// Chmod changes the mode of the named file to mode.
// If the file is a symbolic link, it changes the mode of the link's target.
// If there is an error, it will be of type *PathError.
//
// A different subset of the mode bits are used, depending on the
// operating system.
//
// On Unix, the mode's permission bits, ModeSetuid, ModeSetgid, and
// ModeSticky are used.
//
// On Windows, only the 0200 bit (owner writable) of mode is used; it
// controls whether the file's read-only attribute is set or cleared.
// The other bits are currently unused. For compatibility with Go 1.12
// and earlier, use a non-zero mode. Use mode 0400 for a read-only
// file and 0600 for a readable+writable file.
//
// On Plan 9, the mode's permission bits, ModeAppend, ModeExclusive,
// and ModeTemporary are used.
func (vfs *MountFS) Chmod(name string, mode fs.FileMode) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Chmod(vfsPath, mode)

	return mnt.restoreError(err)
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *MountFS) Chown(name string, uid, gid int) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Chown(vfsPath, uid, gid)

	return mnt.restoreError(err)
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Chtimes(name string, atime, mtime time.Time) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Chtimes(vfsPath, atime, mtime)

	return mnt.restoreError(err)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *MountFS) EvalSymlinks(path string) (string, error) {
	op := "lstat"
	err := error(avfs.ErrPermDenied)

	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
		err = avfs.ErrWinAccessDenied
	}

	return "", &fs.PathError{Op: op, Path: path, Err: err}
}

// Getwd returns a rooted name link corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *MountFS) Getwd() (dir string, err error) {
	cd := vfs.curMnt.toAbsPath(vfs.CurDir())

	return cd, nil
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (vfs *MountFS) Glob(pattern string) (matches []string, err error) {
	matches, err = vfs.VFSFn.Glob(pattern)

	/*
		TODO : restore path if possible or disable function
		for i, m := range matches {
			matches[i] = vfs.fromBasePath(m)
		}
	*/

	return matches, err
}

// Idm returns the identity manager of the file system.
// If the file system does not have an identity manager, avfs.DummyIdm is returned.
func (vfs *MountFS) Idm() avfs.IdentityMgr {
	return vfs.rootFS.Idm()
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *MountFS) Lchown(name string, uid, gid int) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Lchown(vfsPath, uid, gid)

	return mnt.restoreError(err)
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *MountFS) Link(oldname, newname string) error {
	const op = ""

	oldMnt, oldVfsPath := vfs.pathToMount(oldname)
	newMnt, newVfsPath := vfs.pathToMount(newname)

	if oldMnt != newMnt {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrCrossDevLink}
	}

	err := oldMnt.vfs.Link(oldVfsPath, newVfsPath)

	return oldMnt.restoreError(err)
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Lstat(path string) (fs.FileInfo, error) {
	mnt, vfsPath := vfs.pathToMount(path)
	info, err := mnt.vfs.Lstat(vfsPath)

	return info, mnt.restoreError(err)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Mkdir(name string, perm fs.FileMode) error {
	if name == "" {
		err := error(avfs.ErrNoSuchFileOrDir)
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinPathNotFound
		}

		return &fs.PathError{Op: "mkdir", Path: "", Err: err}
	}

	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Mkdir(vfsPath, perm)

	return mnt.restoreError(err)
}

// MkdirAll creates a directory named name,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If name is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *MountFS) MkdirAll(path string, perm fs.FileMode) error {
	absPath, _ := vfs.rootFS.Abs(path)
	mnt := vfs.rootMnt
	mntPos := 0

	pi := avfs.NewPathIterator(vfs, absPath)
	for pi.Next() {
		m, ok := vfs.mounts[pi.LeftPart()]
		if ok {
			vfsPath := absPath[mntPos:pi.End()]

			err := mnt.vfs.MkdirAll(vfsPath, perm)
			if err != nil {
				return mnt.restoreError(err)
			}

			mnt = m
			mntPos = pi.End()
		}
	}

	err := mnt.vfs.MkdirAll(absPath[mntPos:], perm)

	return mnt.restoreError(err)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) OpenFile(name string, flag int, perm fs.FileMode) (avfs.File, error) {
	mnt, vfsPath := vfs.pathToMount(name)

	f, err := mnt.vfs.OpenFile(vfsPath, flag, perm)
	if err != nil {
		return f, mnt.restoreError(err)
	}

	mf := &MountFile{
		vfs:   vfs,
		mount: mnt,
		file:  f,
	}

	return mf, nil
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Readlink(name string) (string, error) {
	const op = "readlink"

	if vfs.OSType() == avfs.OsWindows {
		return "", &fs.PathError{Op: op, Path: name, Err: avfs.ErrWinNotReparsePoint}
	}

	return "", &fs.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Remove(name string) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Remove(vfsPath)

	return mnt.restoreError(err)
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) RemoveAll(path string) error {
	if path == "" {
		// fail silently to retain compatibility with previous behavior of RemoveAll.
		return nil
	}

	mnt, vfsPath := vfs.pathToMount(path)
	err := mnt.vfs.RemoveAll(vfsPath)

	return mnt.restoreError(err)
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *MountFS) Rename(oldname, newname string) error {
	const op = "rename"

	oldMnt, oldVfsPath := vfs.pathToMount(oldname)
	newMnt, newVfsPath := vfs.pathToMount(newname)

	if oldMnt != newMnt {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrCrossDevLink}
	}

	err := oldMnt.vfs.Rename(oldVfsPath, newVfsPath)

	return oldMnt.restoreError(err)
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (vfs *MountFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	return vfs.rootFS.SameFile(fi1, fi2)
}

// SetUMask sets the file mode creation mask.
func (vfs *MountFS) SetUMask(mask fs.FileMode) error {
	return vfs.rootFS.SetUMask(mask)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Stat(path string) (fs.FileInfo, error) {
	mnt, vfsPath := vfs.pathToMount(path)
	info, err := mnt.vfs.Stat(vfsPath)

	return info, mnt.restoreError(err)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (vfs *MountFS) Sub(dir string) (avfs.VFS, error) {
	mnt, vfsPath := vfs.pathToMount(dir)
	info, err := mnt.vfs.Sub(vfsPath)

	return info, mnt.restoreError(err)
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *MountFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	if vfs.OSType() == avfs.OsWindows {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrWinPrivilegeNotHeld}
	}

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrPermDenied}
}

// TempDir returns the default directory to use for temporary files.
//
// On Unix systems, it returns $TMPDIR if non-empty, else /tmp.
// On Windows, it uses GetTempPath, returning the first non-empty
// value from %TMP%, %TEMP%, %USERPROFILE%, or the Windows directory.
// On Plan 9, it returns /tmp.
//
// The directory is neither guaranteed to exist nor have accessible
// permissions.
func (vfs *MountFS) TempDir() string {
	return vfs.rootFS.TempDir()
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *MountFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return vfs.rootFS.ToSysStat(info)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Truncate(name string, size int64) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Truncate(vfsPath, size)

	return mnt.restoreError(err)
}

// UMask returns the file mode creation mask.
func (vfs *MountFS) UMask() fs.FileMode {
	return vfs.rootFS.UMask()
}

// User returns the current user.
func (vfs *MountFS) User() avfs.UserReader {
	return vfs.rootFS.User()
}

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root.
//
// All errors that arise visiting files and directories are filtered by fn:
// see the fs.WalkDirFunc documentation for details.
//
// The files are walked in lexical order, which makes the output deterministic
// but requires WalkDir to read an entire directory into memory before proceeding
// to walk that directory.
//
// WalkDir does not follow symbolic links.
func (vfs *MountFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	err := vfs.VFSFn.WalkDir(root, fn)

	return vfs.rootMnt.restoreError(err)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func (vfs *MountFS) WriteFile(filename string, data []byte, perm fs.FileMode) error {
	err := vfs.VFSFn.WriteFile(filename, data, perm)

	return vfs.rootMnt.restoreError(err)
}
