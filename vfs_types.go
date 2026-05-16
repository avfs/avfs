//
//  Copyright 2023 The AVFS authors
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

package avfs

import (
	"io"
	"io/fs"
	"time"
)

const (
	DefaultDirPerm  = fs.FileMode(0o777) // DefaultDirPerm is the default permission for directories.
	DefaultFilePerm = fs.FileMode(0o666) // DefaultFilePerm is the default permission for files.
	DefaultName     = "Default"          // DefaultName is the default name.
	DefaultVolume   = "C:"               // DefaultVolume is the default volume name for Windows.
	NotImplemented  = "not implemented"  // NotImplemented is the return string of a non-implemented feature.

	// FileModeMask is the bitmask used for permissions.
	FileModeMask = fs.ModePerm | fs.ModeSticky | fs.ModeSetuid | fs.ModeSetgid
)

// Cloner is the interface that wraps the Clone method.
type Cloner interface {
	// Clone returns a shallow copy of the current file system (see MemFs).
	Clone() VFS
}

// ChRooter is the interface that wraps the Chroot method.
type ChRooter interface {
	// Chroot changes the root to that specified in path.
	// If the user has not root privileges avfs.errPermDenied is returned.
	// If there is an error, it will be of type *PathError.
	Chroot(path string) error
}

// DirInfo contains information to create a directory.
type DirInfo struct {
	Path string
	Perm fs.FileMode
}

// File represents a file in the file system.
type File interface {
	fs.File
	fs.ReadDirFile
	io.Reader
	io.ReaderAt
	io.StringWriter
	io.Writer
	io.WriterAt
	io.WriteSeeker

	// Chdir changes the current working directory to the named directory.
	// If there is an error, it will be of type [*PathError].
	Chdir() error

	// Chmod changes the mode of the file to mode.
	// If there is an error, it will be of type [*PathError].
	Chmod(mode fs.FileMode) error

	// Chown changes the numeric uid and gid of the named file.
	// If there is an error, it will be of type [*PathError].
	//
	// On Windows, it always returns the [syscall.EWINDOWS] error, wrapped
	// in [*PathError].
	Chown(uid, gid int) error

	// Fd returns the system file descriptor or handle referencing the open file.
	// If f is closed, the descriptor becomes invalid.
	// If f is garbage collected, a finalizer may close the descriptor,
	// making it invalid; see [runtime.SetFinalizer] for more information on when
	// a finalizer might be run.
	//
	// Do not close the returned descriptor; that could cause a later
	// close of f to close an unrelated descriptor.
	//
	// Fd's behavior differs on some platforms:
	//
	//   - On Unix and Windows, [File.SetDeadline] methods will stop working.
	//   - On Windows, the file descriptor will be disassociated from the
	//     Go runtime I/O completion port if there are no concurrent I/O
	//     operations on the file.
	//
	// For most uses prefer the f.SyscallConn method.
	Fd() uintptr

	// Name returns the name of the file as presented to [Open].
	//
	// It is safe to call Name after [Close].
	Name() string

	// Readdirnames reads the contents of the directory associated with file
	// and returns a slice of up to n names of files in the directory,
	// in directory order. Subsequent calls on the same file will yield
	// further names.
	//
	// If n > 0, Readdirnames returns at most n names. In this case, if
	// Readdirnames returns an empty slice, it will return a non-nil error
	// explaining why. At the end of a directory, the error is [io.EOF].
	//
	// If n <= 0, Readdirnames returns all the names from the directory in
	// a single slice. In this case, if Readdirnames succeeds (reads all
	// the way to the end of the directory), it returns the slice and a
	// nil error. If it encounters an error before the end of the
	// directory, Readdirnames returns the names read until that point and
	// a non-nil error.
	Readdirnames(n int) (names []string, err error)

	// Sync commits the current contents of the file to stable storage.
	// Typically, this means flushing the file system's in-memory copy
	// of recently written data to disk.
	Sync() error

	// Truncate changes the size of the file.
	// It does not change the I/O offset.
	// If there is an error, it will be of type [*PathError].
	Truncate(size int64) error
}

// Namer is the interface that wraps the Name method.
type Namer interface {
	Name() string
}

// SysStater is the interface returned by ToSysStat on all file systems.
type SysStater interface {
	GroupIdentifier
	UserIdentifier
	Nlink() uint64
}

// VolumeManager is the interface that manage volumes for Windows file systems.
type VolumeManager interface {
	// VolumeAdd adds a new volume to a Windows file system.
	// If there is an error, it will be of type *PathError.
	VolumeAdd(name string) error

	// VolumeDelete deletes an existing volume and all its files from a Windows file system.
	// If there is an error, it will be of type *PathError.
	VolumeDelete(name string) error

	// VolumeList returns the volumes of the file system.
	VolumeList() []string
}

// OpenMode defines constants used by OpenFile and CheckPermission functions.
type OpenMode int

const (
	OpenLookup     OpenMode = 0o001     // OpenLookup checks for lookup permission on a directory.
	OpenWrite      OpenMode = 0o002     // OpenWrite opens or checks for write permission.
	OpenRead       OpenMode = 0o004     // OpenRead opens or checks for read permission.
	OpenAppend     OpenMode = 1 << iota // OpenAppend opens a file for appending (os.O_APPEND).
	OpenCreate                          // OpenCreate creates a file (os.O_CREATE).
	OpenCreateExcl                      // OpenCreateExcl creates a non existing file (os.O_EXCL).
	OpenTruncate                        // OpenTruncate truncates a file (os.O_TRUNC).
	OpenDir                             // OpenDir opens a directory (syscall.O_DIRECTORY) returns error otherwise.
)

// IOFS is the virtual file system interface implementing io/fs interfaces.
type IOFS interface {
	VFSBase
	fs.FS
	fs.GlobFS
	fs.ReadDirFS
	fs.ReadFileFS
	fs.StatFS
	fs.SubFS
}

// Typer is the interface that wraps the Type method.
type Typer interface {
	// Type returns the type of the fileSystem or Identity manager.
	Type() string
}

// VFS is the virtual file system interface.
// Any simulated or real file system should implement this interface.
type VFS interface {
	VFSBase

	// Open opens the named file for reading. If successful, methods on
	// the returned file can be used for reading; the associated file
	// descriptor has mode [O_RDONLY].
	// If there is an error, it will be of type [*PathError].
	Open(name string) (File, error)

	// Sub returns an FS corresponding to the subtree rooted at dir.
	Sub(dir string) (VFS, error)
}

// VFSBase regroups the common methods to VFS and IOFS.
type VFSBase interface {
	Featurer
	IdmMgr
	Namer
	OSTyper
	Typer
	UMasker
	VFSPath
	VFSUserDir

	// Base returns the last element of path.
	// Trailing path separators are removed before extracting the last element.
	// If the path is empty, Base returns ".".
	// If the path consists entirely of separators, Base returns a single separator.
	Base(path string) string

	// Chdir changes the current working directory to the named directory.
	// If there is an error, it will be of type [*PathError].
	Chdir(dir string) error

	// Chmod changes the mode of the named file to mode.
	// If the file is a symbolic link, it changes the mode of the link's target.
	// If there is an error, it will be of type [*PathError].
	//
	// A different subset of the mode bits are used, depending on the
	// operating system.
	//
	// On Unix, the mode's permission bits, [ModeSetuid], [ModeSetgid], and
	// [ModeSticky] are used.
	//
	// On Windows, only the 0o200 bit (owner writable) of mode is used; it
	// controls whether the file's read-only attribute is set or cleared.
	// The other bits are currently unused. For compatibility with Go 1.12
	// and earlier, use a non-zero mode. Use mode 0o400 for a read-only
	// file and 0o600 for a readable+writable file.
	//
	// On Plan 9, the mode's permission bits, [ModeAppend], [ModeExclusive],
	// and [ModeTemporary] are used.
	Chmod(name string, mode fs.FileMode) error

	// Chown changes the numeric uid and gid of the named file.
	// If the file is a symbolic link, it changes the uid and gid of the link's target.
	// A uid or gid of -1 means to not change that value.
	// If there is an error, it will be of type [*PathError].
	//
	// On Windows or Plan 9, Chown always returns the [syscall.EWINDOWS] or
	// [syscall.EPLAN9] error, wrapped in [*PathError].
	Chown(name string, uid, gid int) error

	// Chtimes changes the access and modification times of the named
	// file, similar to the Unix utime() or utimes() functions.
	// A zero [time.Time] value will leave the corresponding file time unchanged.
	//
	// The underlying filesystem may truncate or round the values to a
	// less precise time unit.
	// If there is an error, it will be of type [*PathError].
	Chtimes(name string, atime, mtime time.Time) error

	// Create creates or truncates the named file. If the file already exists,
	// it is truncated. If the file does not exist, it is created with mode 0o666
	// (before umask). If successful, methods on the returned File can
	// be used for I/O; the associated file descriptor has mode [O_RDWR].
	// The directory containing the file must already exist.
	// If there is an error, it will be of type [*PathError].
	Create(name string) (File, error)

	// CreateTemp creates a new temporary file in the directory dir,
	// opens the file for reading and writing, and returns the resulting file.
	// The filename is generated by taking pattern and adding a random string to the end.
	// If pattern includes a "*", the random string replaces the last "*".
	// The file is created with mode 0o600 (before umask).
	// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by [TempDir].
	// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
	// The caller can use the file's Name method to find the pathname of the file.
	// It is the caller's responsibility to remove the file when it is no longer needed.
	CreateTemp(dir, pattern string) (File, error)

	// EvalSymlinks returns the path name after the evaluation of any symbolic
	// links.
	// If path is relative the result will be relative to the current directory,
	// unless one of the components is an absolute symbolic link.
	// EvalSymlinks calls [Clean] on the result.
	EvalSymlinks(path string) (string, error)

	// Glob returns the names of all files matching pattern or nil
	// if there is no matching file. The syntax of patterns is the same
	// as in [Match]. The pattern may describe hierarchical names such as
	// /usr/*/bin/ed (assuming the [Separator] is '/').
	//
	// Glob ignores file system errors such as I/O errors reading directories.
	// The only possible returned error is [ErrBadPattern], when pattern
	// is malformed.
	Glob(pattern string) (matches []string, err error)

	// Idm returns the identity manager of the file system.
	// If the file system does not have an identity manager, avfs.DummyIdm is returned.
	Idm() IdentityMgr

	// Lchown changes the numeric uid and gid of the named file.
	// If the file is a symbolic link, it changes the uid and gid of the link itself.
	// If there is an error, it will be of type [*PathError].
	//
	// On Windows, it always returns the [syscall.EWINDOWS] error, wrapped
	// in [*PathError].
	Lchown(name string, uid, gid int) error

	// Link creates newname as a hard link to the oldname file.
	// If there is an error, it will be of type *LinkError.
	Link(oldname, newname string) error

	// Lstat returns a [FileInfo] describing the named file.
	// If the file is a symbolic link, the returned FileInfo
	// describes the symbolic link. Lstat makes no attempt to follow the link.
	// If there is an error, it will be of type [*PathError].
	//
	// On Windows, if the file is a reparse point that is a surrogate for another
	// named entity (such as a symbolic link or mounted folder), the returned
	// FileInfo describes the reparse point, and makes no attempt to resolve it.
	Lstat(name string) (fs.FileInfo, error)

	// Mkdir creates a new directory with the specified name and permission
	// bits (before umask).
	// If there is an error, it will be of type [*PathError].
	Mkdir(name string, perm fs.FileMode) error

	// MkdirAll creates a directory named path,
	// along with any necessary parents, and returns nil,
	// or else returns an error.
	// The permission bits perm (before umask) are used for all
	// directories that MkdirAll creates.
	// If path is already a directory, MkdirAll does nothing
	// and returns nil.
	MkdirAll(path string, perm fs.FileMode) error

	// MkdirTemp creates a new temporary directory in the directory dir
	// and returns the pathname of the new directory.
	// The new directory's name is generated by adding a random string to the end of pattern.
	// If pattern includes a "*", the random string replaces the last "*" instead.
	// The directory is created with mode 0o700 (before umask).
	// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
	// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
	// It is the caller's responsibility to remove the directory when it is no longer needed.
	MkdirTemp(dir, pattern string) (string, error)

	// OpenFile is the generalized open call; most users will use Open
	// or Create instead. It opens the named file with specified flag
	// ([O_RDONLY] etc.). If the file does not exist, and the [O_CREATE] flag
	// is passed, it is created with mode perm (before umask);
	// the containing directory must exist. If successful,
	// methods on the returned File can be used for I/O.
	// If there is an error, it will be of type [*PathError].
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)

	// ReadDir reads the named directory,
	// returning all its directory entries sorted by filename.
	// If an error occurs reading the directory,
	// ReadDir returns the entries it was able to read before the error,
	// along with the error.
	ReadDir(name string) ([]fs.DirEntry, error)

	// ReadFile reads the named file and returns the contents.
	// A successful call returns err == nil, not err == EOF.
	// Because ReadFile reads the whole file, it does not treat an EOF from Read
	// as an error to be reported.
	ReadFile(name string) ([]byte, error)

	// Readlink returns the destination of the named symbolic link.
	// If there is an error, it will be of type [*PathError].
	//
	// If the link destination is relative, Readlink returns the relative path
	// without resolving it to an absolute one.
	Readlink(name string) (string, error)

	// Remove removes the named file or (empty) directory.
	// If there is an error, it will be of type [*PathError].
	Remove(name string) error

	// RemoveAll removes path and any children it contains.
	// It removes everything it can but returns the first error
	// it encounters. If the path does not exist, RemoveAll
	// returns nil (no error).
	// If there is an error, it will be of type [*PathError].
	RemoveAll(path string) error

	// Rename renames (moves) oldpath to newpath.
	// If newpath already exists and is not a directory, Rename replaces it.
	// If newpath already exists and is a directory, Rename returns an error.
	// OS-specific restrictions may apply when oldpath and newpath are in different directories.
	// Even within the same directory, on non-Unix platforms Rename is not an atomic operation.
	// If there is an error, it will be of type *LinkError.
	Rename(oldpath, newpath string) error

	// SameFile reports whether fi1 and fi2 describe the same file.
	// For example, on Unix this means that the device and inode fields
	// of the two underlying structures are identical; on other systems
	// the decision may be based on the path names.
	// SameFile only applies to results returned by this package's [Stat].
	// It returns false in other cases.
	SameFile(fi1, fi2 fs.FileInfo) bool

	// Stat returns a [FileInfo] describing the named file.
	// If there is an error, it will be of type [*PathError].
	Stat(name string) (fs.FileInfo, error)

	// Symlink creates newname as a symbolic link to oldname.
	// On Windows, a symlink to a non-existent oldname creates a file symlink;
	// if oldname is later created as a directory the symlink will not work.
	// If there is an error, it will be of type *LinkError.
	Symlink(oldname, newname string) error

	// Truncate changes the size of the named file.
	// If the file is a symbolic link, it changes the size of the link's target.
	// If there is an error, it will be of type [*PathError].
	Truncate(name string, size int64) error

	// WalkDir walks the file tree rooted at root, calling fn for each file or
	// directory in the tree, including root.
	//
	// All errors that arise visiting files and directories are filtered by fn:
	// see the [fs.WalkDirFunc] documentation for details.
	//
	// The files are walked in lexical order, which makes the output deterministic
	// but requires WalkDir to read an entire directory into memory before proceeding
	// to walk that directory.
	//
	// WalkDir does not follow symbolic links.
	//
	// WalkDir calls fn with paths that use the separator character appropriate
	// for the operating system. This is unlike [io/fs.WalkDir], which always
	// uses slash separated paths.
	WalkDir(root string, fn fs.WalkDirFunc) error

	// WriteFile writes data to the named file, creating it if necessary.
	// If the file does not exist, WriteFile creates it with permissions perm (before umask);
	// otherwise WriteFile truncates it before writing, without changing permissions.
	// Since WriteFile requires multiple system calls to complete, a failure mid-operation
	// can leave the file in a partially written state.
	WriteFile(name string, data []byte, perm fs.FileMode) error
}
