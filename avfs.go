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

// Package avfs defines interfaces, errors types used by all file systems implementations.
package avfs

import (
	"io"
	"io/fs"
	"time"
)

const (
	DefaultDirPerm  = fs.FileMode(0o777) // DefaultDirPerm is the default permission for directories.
	DefaultFilePerm = fs.FileMode(0o666) // DefaultFilePerm is the default permission for files.
	DefaultVolume   = "C:"               // DefaultVolume is the default volume name for Windows.
	MaxInt          = int(^uint(0) >> 1)
	NotImplemented  = "not implemented" // NotImplemented is the return string of a non-implemented feature.

	// FileModeMask is the bitmask used for permissions.
	FileModeMask = fs.ModePerm | fs.ModeSticky | fs.ModeSetuid | fs.ModeSetgid
)

// Features defines the set of features available on a file system.
type Features uint64

//go:generate stringer -type Features -trimprefix Feat -bitmask -output avfs_features.go

const (
	// FeatBasicFs indicates that the file system implements all basic functions.
	FeatBasicFs Features = 1 << iota

	// FeatChroot indicates that the file system supports Chroot.
	FeatChroot

	// FeatChownUser indicates that a non privileged user can use Chown.
	FeatChownUser

	// FeatMainDirs indicates that the main directories of the filesystem (/home, /root and /tmp) are present.
	FeatMainDirs

	// FeatHardlink indicates that the file system supports hard links (link(), readlink() functions).
	FeatHardlink

	// FeatIdentityMgr indicates that the file system features and identity manager and supports multiple users.
	FeatIdentityMgr

	// FeatReadOnly indicates that the file system is a read only file system (see RoFs).
	FeatReadOnly

	// FeatReadOnlyIdm indicates that the identity manager is a read only (see OsIdm).
	FeatReadOnlyIdm

	// FeatRealFS indicates that the file system is a real one, not emulated (see OsFS).
	FeatRealFS

	// FeatSymlink indicates that the file system supports symbolic links (symlink(), evalSymlink() functions).
	FeatSymlink
)

// OSType defines the operating system type.
type OSType uint16

//go:generate stringer -type OSType -linecomment -output avfs_ostype.go

const (
	OsLinux   OSType = iota // Linux
	OsWindows               // Windows
	OsDarwin                // Darwin
	OsUnknown               // Unknown
)

// OpenMode defines constants used by OpenFile and CheckPermission functions.
type OpenMode uint16

const (
	OpenLookup     OpenMode = 0o001     // OpenLookup checks for lookup permission on a directory.
	OpenWrite      OpenMode = 0o002     // OpenWrite opens or checks for write permission.
	OpenRead       OpenMode = 0o004     // OpenRead opens or checks for read permission.
	OpenAppend     OpenMode = 1 << iota // OpenAppend opens a file for appending (os.O_APPEND).
	OpenCreate                          // OpenCreate creates a file (os.O_CREATE).
	OpenCreateExcl                      // OpenCreateExcl creates a non existing file (os.O_EXCL).
	OpenTruncate                        // OpenTruncate truncates a file (os.O_TRUNC).
)

// VFS is the virtual file system interface.
// Any simulated or real file system should implement this interface.
type VFS interface {
	BaseVFS
	ChRooter

	// Open opens the named file for reading. If successful, methods on
	// the returned file can be used for reading; the associated file
	// descriptor has mode O_RDONLY.
	// If there is an error, it will be of type *PathError.
	Open(name string) (File, error)
}

// IOFS is the virtual file system interface implementing io/fs interfaces.
type IOFS interface {
	BaseVFS
	fs.FS
	fs.GlobFS
	fs.ReadDirFS
	fs.ReadFileFS
	fs.StatFS
	fs.SubFS
}

// BaseVFS regroups the common methods to VFS and IOFS.
type BaseVFS interface {
	Featurer
	Namer
	Typer

	// Abs returns an absolute representation of path.
	// If the path is not absolute it will be joined with the current
	// working directory to turn it into an absolute path. The absolute
	// path name for a given file is not guaranteed to be unique.
	// Abs calls Clean on the result.
	Abs(path string) (string, error)

	// Base returns the last element of path.
	// Trailing path separators are removed before extracting the last element.
	// If the path is empty, Base returns ".".
	// If the path consists entirely of separators, Base returns a single separator.
	Base(path string) string

	// Chdir changes the current working directory to the named directory.
	// If there is an error, it will be of type *PathError.
	Chdir(dir string) error

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
	Chmod(name string, mode fs.FileMode) error

	// Chown changes the numeric uid and gid of the named file.
	// If the file is a symbolic link, it changes the uid and gid of the link's target.
	// A uid or gid of -1 means to not change that value.
	// If there is an error, it will be of type *PathError.
	//
	// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
	// EPLAN9 error, wrapped in *PathError.
	Chown(name string, uid, gid int) error

	// Chtimes changes the access and modification times of the named
	// file, similar to the Unix utime() or utimes() functions.
	//
	// The underlying file system may truncate or round the values to a
	// less precise time unit.
	// If there is an error, it will be of type *PathError.
	Chtimes(name string, atime, mtime time.Time) error

	// Clean returns the shortest path name equivalent to path
	// by purely lexical processing. It applies the following rules
	// iteratively until no further processing can be done:
	//
	//	1. Replace multiple Separator elements with a single one.
	//	2. Eliminate each . path name element (the current directory).
	//	3. Eliminate each inner .. path name element (the parent directory)
	//	   along with the non-.. element that precedes it.
	//	4. Eliminate .. elements that begin a rooted path:
	//	   that is, replace "/.." by "/" at the beginning of a path,
	//	   assuming Separator is '/'.
	//
	// The returned path ends in a slash only if it represents a root directory,
	// such as "/" on Unix or `C:\` on Windows.
	//
	// Finally, any occurrences of slash are replaced by Separator.
	//
	// If the result of this process is an empty string, Clean
	// returns the string ".".
	//
	// See also Rob Pike, ``Lexical File Names in Plan 9 or
	// Getting Dot-Dot Right,''
	// https://9p.io/sys/doc/lexnames.html
	Clean(path string) string

	// Create creates the named file with mode 0666 (before umask), truncating
	// it if it already exists. If successful, methods on the returned
	// File can be used for I/O; the associated file descriptor has mode
	// O_RDWR.
	// If there is an error, it will be of type *PathError.
	Create(name string) (File, error)

	// CreateTemp creates a new temporary file in the directory dir,
	// opens the file for reading and writing, and returns the resulting file.
	// The filename is generated by taking pattern and adding a random string to the end.
	// If pattern includes a "*", the random string replaces the last "*".
	// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by TempDir.
	// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
	// The caller can use the file's Name method to find the pathname of the file.
	// It is the caller's responsibility to remove the file when it is no longer needed.
	CreateTemp(dir, pattern string) (File, error)

	// Dir returns all but the last element of path, typically the path's directory.
	// After dropping the final element, Dir calls Clean on the path and trailing
	// slashes are removed.
	// If the path is empty, Dir returns ".".
	// If the path consists entirely of separators, Dir returns a single separator.
	// The returned path does not end in a separator unless it is the root directory.
	Dir(path string) string

	// EvalSymlinks returns the path name after the evaluation of any symbolic
	// links.
	// If path is relative the result will be relative to the current directory,
	// unless one of the components is an absolute symbolic link.
	// EvalSymlinks calls Clean on the result.
	EvalSymlinks(path string) (string, error)

	// FromSlash returns the result of replacing each slash ('/') character
	// in path with a separator character. Multiple slashes are replaced
	// by multiple separators.
	FromSlash(path string) string

	// Getwd returns a rooted path name corresponding to the
	// current directory. If the current directory can be
	// reached via multiple paths (due to symbolic links),
	// Getwd may return any one of them.
	Getwd() (dir string, err error)

	// Glob returns the names of all files matching pattern or nil
	// if there is no matching file. The syntax of patterns is the same
	// as in Match. The pattern may describe hierarchical names such as
	// /usr/*/bin/ed (assuming the Separator is '/').
	//
	// Glob ignores file system errors such as I/O errors reading directories.
	// The only possible returned error is ErrBadPattern, when pattern
	// is malformed.
	Glob(pattern string) (matches []string, err error)

	// Idm returns the identity manager of the file system.
	// if the file system does not have an identity manager, avfs.DummyIdm is returned.
	Idm() IdentityMgr

	// IsAbs reports whether the path is absolute.
	IsAbs(path string) bool

	// IsPathSeparator reports whether c is a directory separator character.
	IsPathSeparator(c uint8) bool

	// Join joins any number of path elements into a single path, adding a
	// separating slash if necessary. The result is Cleaned; in particular,
	// all empty strings are ignored.
	Join(elem ...string) string

	// Lchown changes the numeric uid and gid of the named file.
	// If the file is a symbolic link, it changes the uid and gid of the link itself.
	// If there is an error, it will be of type *PathError.
	//
	// On Windows, it always returns the syscall.EWINDOWS error, wrapped
	// in *PathError.
	Lchown(name string, uid, gid int) error

	// Link creates newname as a hard link to the oldname file.
	// If there is an error, it will be of type *LinkError.
	Link(oldname, newname string) error

	// Lstat returns a FileInfo describing the named file.
	// If the file is a symbolic link, the returned FileInfo
	// describes the symbolic link. Lstat makes no attempt to follow the link.
	// If there is an error, it will be of type *PathError.
	Lstat(name string) (fs.FileInfo, error)

	// Match reports whether name matches the shell file name pattern.
	// The pattern syntax is:
	//
	//	pattern:
	//		{ term }
	//	term:
	//		'*'         matches any sequence of non-Separator characters
	//		'?'         matches any single non-Separator character
	//		'[' [ '^' ] { character-range } ']'
	//		            character class (must be non-empty)
	//		c           matches character c (c != '*', '?', '\\', '[')
	//		'\\' c      matches character c
	//
	//	character-range:
	//		c           matches character c (c != '\\', '-', ']')
	//		'\\' c      matches character c
	//		lo '-' hi   matches character c for lo <= c <= hi
	//
	// Match requires pattern to match all of name, not just a substring.
	// The only possible returned error is ErrBadPattern, when pattern
	// is malformed.
	//
	// On Windows, escaping is disabled. Instead, '\\' is treated as
	// path separator.
	//
	Match(pattern, name string) (matched bool, err error)

	// Mkdir creates a new directory with the specified name and permission
	// bits (before umask).
	// If there is an error, it will be of type *PathError.
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
	// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
	// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
	// It is the caller's responsibility to remove the directory when it is no longer needed.
	MkdirTemp(dir, pattern string) (string, error)

	// OpenFile is the generalized open call; most users will use Open
	// or Create instead. It opens the named file with specified flag
	// (O_RDONLY etc.) and perm (before umask), if applicable. If successful,
	// methods on the returned File can be used for I/O.
	// If there is an error, it will be of type *PathError.
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)

	// OSType returns the operating system type of the file system.
	OSType() OSType

	// PathSeparator return the OS-specific path separator.
	PathSeparator() uint8

	// ReadDir reads the named directory,
	// returning all its directory entries sorted by filename.
	// If an error occurs reading the directory,
	// ReadDir returns the entries it was able to read before the error,
	// along with the error.
	ReadDir(name string) ([]fs.DirEntry, error)

	// ReadFile reads the file named by filename and returns the contents.
	// A successful call returns err == nil, not err == EOF. Because ReadFile
	// reads the whole file, it does not treat an EOF from Read as an error
	// to be reported.
	ReadFile(filename string) ([]byte, error)

	// Readlink returns the destination of the named symbolic link.
	// If there is an error, it will be of type *PathError.
	Readlink(name string) (string, error)

	// Rel returns a relative path that is lexically equivalent to targpath when
	// joined to basepath with an intervening separator. That is,
	// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
	// On success, the returned path will always be relative to basepath,
	// even if basepath and targpath share no elements.
	// An error is returned if targpath can't be made relative to basepath or if
	// knowing the current working directory would be necessary to compute it.
	// Rel calls Clean on the result.
	Rel(basepath, targpath string) (string, error)

	// Remove removes the named file or (empty) directory.
	// If there is an error, it will be of type *PathError.
	Remove(name string) error

	// RemoveAll removes path and any children it contains.
	// It removes everything it can but returns the first error
	// it encounters. If the path does not exist, RemoveAll
	// returns nil (no error).
	RemoveAll(path string) error

	// Rename renames (moves) oldpath to newpath.
	// If newpath already exists and is not a directory, Rename replaces it.
	// OS-specific restrictions may apply when oldpath and newpath are in different directories.
	// If there is an error, it will be of type *LinkError.
	Rename(oldpath, newpath string) error

	// SameFile reports whether fi1 and fi2 describe the same file.
	// For example, on Unix this means that the device and inode fields
	// of the two underlying structures are identical; on other systems
	// the decision may be based on the path names.
	// SameFile only applies to results returned by this package's Stat.
	// It returns false in other cases.
	SameFile(fi1, fi2 fs.FileInfo) bool

	// SetUMask sets the file mode creation mask.
	SetUMask(mask fs.FileMode)

	// SetUser sets and returns the current user.
	// If the user is not found, the returned error is of type UnknownUserError.
	SetUser(name string) (UserReader, error)

	// Stat returns a FileInfo describing the named file.
	// If there is an error, it will be of type *PathError.
	Stat(name string) (fs.FileInfo, error)

	// Split splits path immediately following the final Separator,
	// separating it into a directory and file name component.
	// If there is no Separator in path, Split returns an empty dir
	// and file set to path.
	// The returned values have the property that path = dir+file.
	Split(path string) (dir, file string)

	// Symlink creates newname as a symbolic link to oldname.
	// If there is an error, it will be of type *LinkError.
	Symlink(oldname, newname string) error

	// TempDir returns the default directory to use for temporary files.
	//
	// On Unix systems, it returns $TMPDIR if non-empty, else /tmp.
	// On Windows, it uses GetTempPath, returning the first non-empty
	// value from %TMP%, %TEMP%, %USERPROFILE%, or the Windows directory.
	// On Plan 9, it returns /tmp.
	//
	// The directory is neither guaranteed to exist nor have accessible
	// permissions.
	TempDir() string

	// ToSlash returns the result of replacing each separator character
	// in path with a slash ('/') character. Multiple separators are
	// replaced by multiple slashes.
	ToSlash(path string) string

	// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
	ToSysStat(info fs.FileInfo) SysStater

	// Truncate changes the size of the named file.
	// If the file is a symbolic link, it changes the size of the link's target.
	// If there is an error, it will be of type *PathError.
	Truncate(name string, size int64) error

	// UMask returns the file mode creation mask.
	UMask() fs.FileMode

	// User returns the current user.
	// if the file system does not have a current user, the user avfs.DefaultUser is returned.
	User() UserReader

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
	WalkDir(root string, fn fs.WalkDirFunc) error

	// WriteFile writes data to a file named by filename.
	// If the file does not exist, WriteFile creates it with permissions perm;
	// otherwise WriteFile truncates it before writing.
	WriteFile(filename string, data []byte, perm fs.FileMode) error
}

// ChRooter is the interface that wraps the Chroot method.
type ChRooter interface {
	// Chroot changes the root to that specified in path.
	// If the user has not root privileges avfs.errPermDenied is returned.
	// If there is an error, it will be of type *PathError.
	Chroot(path string) error
}

// Cloner is the interface that wraps the Clone method.
type Cloner interface {
	// Clone returns a shallow copy of the current file system (see MemFs).
	Clone() VFS
}

// Featurer is the interface that wraps the Features and HasFeature methods.
type Featurer interface {
	// Features returns the set of features provided by the file system or identity manager.
	Features() Features

	// HasFeature returns true if the file system or identity manager provides a given feature.
	HasFeature(feature Features) bool
}

// Namer is the interface that wraps the Name method.
type Namer interface {
	Name() string
}

// Typer is the interface that wraps the Type method.
type Typer interface {
	// Type returns the type of the fileSystem or Identity manager.
	Type() string
}

// File represents a file in the file system.
type File interface {
	fs.File
	fs.ReadDirFile
	io.ReaderAt
	io.StringWriter
	io.WriterAt
	io.WriteSeeker

	// Chdir changes the current working directory to the file,
	// which must be a directory.
	// If there is an error, it will be of type *PathError.
	Chdir() error

	// Chmod changes the mode of the file to mode.
	// If there is an error, it will be of type *PathError.
	Chmod(mode fs.FileMode) error

	// Chown changes the numeric uid and gid of the named file.
	// If there is an error, it will be of type *PathError.
	//
	// On Windows, it always returns the syscall.EWINDOWS error, wrapped
	// in *PathError.
	Chown(uid, gid int) error

	// Fd returns the integer Unix file descriptor referencing the open file.
	// The file descriptor is valid only until f.Close is called or f is garbage collected.
	// On Unix systems this will cause the SetDeadline methods to stop working.
	Fd() uintptr

	// Name returns the name of the file as presented to Open.
	Name() string

	// Readdirnames reads and returns a slice of names from the directory f.
	//
	// If n > 0, Readdirnames returns at most n names. In this case, if
	// Readdirnames returns an empty slice, it will return a non-nil error
	// explaining why. At the end of a directory, the error is io.EOF.
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
	// If there is an error, it will be of type *PathError.
	Truncate(size int64) error
}

// IdentityMgr interface manages identities (users and groups).
type IdentityMgr interface {
	Featurer
	Typer

	// AdminGroup returns the administrator (root) group.
	AdminGroup() GroupReader

	// AdminUser returns the administrator (root) user.
	AdminUser() UserReader

	// GroupAdd adds a new group.
	// If the group already exists, the returned error is of type AlreadyExistsGroupError.
	GroupAdd(name string) (GroupReader, error)

	// GroupDel deletes an existing group.
	// If the group is not found, the returned error is of type UnknownGroupError.
	GroupDel(name string) error

	// LookupGroup looks up a group by name.
	// If the group is not found, the returned error is of type UnknownGroupError.
	LookupGroup(name string) (GroupReader, error)

	// LookupGroupId looks up a group by groupid.
	// If the group is not found, the returned error is of type UnknownGroupIdError.
	LookupGroupId(gid int) (GroupReader, error)

	// LookupUser looks up a user by username.
	// If the user cannot be found, the returned error is of type UnknownUserError.
	LookupUser(name string) (UserReader, error)

	// LookupUserId looks up a user by userid.
	// If the user cannot be found, the returned error is of type UnknownUserIdError.
	LookupUserId(uid int) (UserReader, error)

	// UserAdd adds a new user.
	// If the user already exists, the returned error is of type AlreadyExistsUserError.
	UserAdd(name, groupName string) (UserReader, error)

	// UserDel deletes an existing user.
	UserDel(name string) error
}

// GroupIdentifier is the interface that wraps the Gid method.
type GroupIdentifier interface {
	// Gid returns the primary group id.
	Gid() int
}

// GroupReader interface reads group information.
type GroupReader interface {
	GroupIdentifier
	Namer
}

// UserIdentifier is the interface that wraps the Uid method.
type UserIdentifier interface {
	// Uid returns the user id.
	Uid() int
}

// UserReader reads user information.
type UserReader interface {
	GroupIdentifier
	UserIdentifier
	Namer

	// IsAdmin returns true if the user has administrator (root) privileges.
	IsAdmin() bool
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
