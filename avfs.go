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
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"syscall"
	"time"
)

const (
	// Avfs is the name of the framework.
	Avfs = "avfs"

	// PathSeparator is defined as a forward slash for all systems.
	PathSeparator = '/'

	// DefaultDirPerm is the default permission for directories.
	DefaultDirPerm = 0o755

	// DefaultFilePerm is the default permission for files.
	DefaultFilePerm = 0o644

	// HomeDirPerm is the default permission for home directories.
	HomeDirPerm = 0o700

	// FileModeMask is the bitmask used for permissions (see os.Chmod() comment).
	FileModeMask = os.ModePerm | os.ModeSticky | os.ModeSetuid | os.ModeSetgid

	// HomeDir is the home directory.
	HomeDir = "/home"

	// RootDir is the root directory.
	RootDir = "/root"

	// TmpDir is the tmp directory.
	TmpDir = "/tmp"

	// UsrRoot is the root user.
	UsrRoot = "root"

	// NotImplemented is the return string of a non implemented feature.
	NotImplemented = "not implemented"
)

// Errors on linux and Windows operating systems.
// Most of the errors below can be found there :
// https://github.com/torvalds/linux/blob/master/tools/include/uapi/asm-generic/errno-base.h
const (
	// ErrBadFileDesc is the error Bad file descriptor.
	ErrBadFileDesc = syscall.EBADF

	// ErrCrossDevLink is the error Cross-device link.
	// ErrCrossDevLink = syscall.EXDEV

	// ErrDirNotEmpty is the error Directory not empty.
	ErrDirNotEmpty = syscall.ENOTEMPTY

	// ErrFileExists is the error File exists.
	ErrFileExists = syscall.EEXIST

	// ErrIsADirectory is the error Is a directory.
	ErrIsADirectory = syscall.EISDIR

	// ErrNoSuchFileOrDir is the error No such file or directory.
	ErrNoSuchFileOrDir = syscall.ENOENT

	// ErrNotADirectory is the error Not a directory.
	ErrNotADirectory = syscall.ENOTDIR

	// ErrOpNotPermitted is the error Operation not permitted.
	ErrOpNotPermitted = syscall.EPERM

	// ErrPermDenied is the error Permission denied.
	ErrPermDenied = syscall.EACCES

	// ErrTooManySymlinks is the error Too many levels of symbolic links.
	ErrTooManySymlinks = syscall.ELOOP
)

// Errors on windows operating systems only.
const (
	// ErrWinInvalidHandle is the error The handle is invalid.
	ErrWinInvalidHandle = syscall.Errno(0x6)

	// ErrWinPathNotFound is the error The system cannot find the path specified (syscall.ERROR_PATH_NOT_FOUND).
	ErrWinPathNotFound = syscall.Errno(0x3)

	// ErrWinNotSupported is the error Not supported by windows (syscall.EWINDOWS).
	ErrWinNotSupported = syscall.Errno(0x20000082)

	// ErrWinDirNameInvalid is the error The directory name is invalid.
	ErrWinDirNameInvalid = syscall.Errno(0x10B)

	// ErrWinAccessDenied is the error Access is denied.
	ErrWinAccessDenied = syscall.Errno(0x5)

	// ErrWinFileExists is the error The file exists (syscall.ERROR_FILE_EXISTS).
	ErrWinFileExists = syscall.Errno(80)
)

var (
	// ErrNegativeOffset is the Error negative offset.
	ErrNegativeOffset = errors.New("negative offset")

	// ErrFileClosing is returned when a file descriptor is used after it
	// has been closed.
	ErrFileClosing = errors.New("use of closed file")
)

// AlreadyExistsGroupError is returned when the group name already exists.
type AlreadyExistsGroupError string

func (e AlreadyExistsGroupError) Error() string {
	return "group: group " + string(e) + " already exists"
}

// AlreadyExistsUserError is returned when the user name already exists.
type AlreadyExistsUserError string

func (e AlreadyExistsUserError) Error() string {
	return "user: user " + string(e) + " already exists"
}

// UnknownError is returned when there is an unknown error.
type UnknownError string

func (e UnknownError) Error() string {
	return "unknown error " + reflect.TypeOf(e).String() + " : '" + string(e) + "'"
}

// UnknownGroupError is returned by LookupGroup when a group cannot be found.
type UnknownGroupError string

func (e UnknownGroupError) Error() string {
	return "group: unknown group " + string(e)
}

// UnknownGroupIdError is returned by LookupGroupId when a group cannot be found.
type UnknownGroupIdError int

func (e UnknownGroupIdError) Error() string {
	return "group: unknown groupid " + strconv.Itoa(int(e))
}

// UnknownUserError is returned by Lookup when a user cannot be found.
type UnknownUserError string

func (e UnknownUserError) Error() string {
	return "user: unknown user " + string(e)
}

// UnknownUserIdError is returned by LookupUserId when a user cannot be found.
type UnknownUserIdError int

func (e UnknownUserIdError) Error() string {
	return "user: unknown userid " + strconv.Itoa(int(e))
}

// Feature defines the list of features available on a file system.
type Feature uint64

const (
	// FeatAbsPath indicates that all paths passed as parameters are absolute paths.
	FeatAbsPath Feature = 1 << iota

	// FeatBasicFs indicates that the file system implements all basic functions.
	FeatBasicFs

	// FeatChroot indicates that the file system supports Chroot.
	FeatChroot

	// FeatClonable indicates if a file system can be cloned.
	FeatClonable

	// FeatMainDirs indicates that the main directories of the filesystem (/home, /root and /tmp) are present.
	FeatMainDirs

	// FeatHardlink indicates that the file system supports hard links (link(), readlink() functions).
	FeatHardlink

	// FeatIdentityMgr indicates that the file system features and identity manager and supports multiple users.
	// (Chown(), User(), CurrentUser(), ... functions).
	FeatIdentityMgr

	// FeatReadOnly indicates that the file system is a read only file system (see RoFs).
	FeatReadOnly

	// FeatSymlink indicates that the file system supports symbolic links (symlink(), evalSymlink() functions).
	FeatSymlink
)

// OSType defines the operating system type.
type OSType uint8

const (
	// OsDarwin is the Darwin operating system.
	OsDarwin OSType = iota + 1

	// OsLinux is the Linux operating system.
	OsLinux

	// OsUnknown is an unknown operating system.
	OsUnknown

	// OsWindows is the Windows operating system.
	OsWindows
)

// WantMode defines the permissions to check for CheckPermission() function.
type WantMode uint8

const (
	// WantLookup checks for lookup permission on a directory.
	WantLookup WantMode = 0o001

	// WantWrite checks for write permission.
	WantWrite WantMode = 0o002

	// WantRead checks for read permission.
	WantRead WantMode = 0o004

	// WantRWX checks for all permissions.
	WantRWX WantMode = 0o007
)

// Fs is the file system interface.
// Any simulated or real file system should implement this interface.
type Fs interface {
	BasicFs
	ChDirer
	ChModer
	ChOwner
	ChRooter
	ChTimer
	Cloner
	IdentityMgr
	HardLinker
	Namer
	OSTyper
	Pather
	SymLinker
	UMasker
	UserConnecter
}

// BasicFs is the basic file system interface (no hard links, symbolic links, users, permissions and chroot).
// Any simulated or real file system should implement this interface.
type BasicFs interface {
	// Create creates the named file with mode 0666 (before umask), truncating
	// it if it already exists. If successful, methods on the returned
	// File can be used for I/O; the associated file descriptor has mode
	// O_RDWR.
	// If there is an error, it will be of type *PathError.
	Create(name string) (File, error)

	// GetTempDir returns the default directory to use for temporary files.
	//
	// On Unix systems, it returns $TMPDIR if non-empty, else /tmp.
	// On Windows, it uses GetTempPath, returning the first non-empty
	// value from %TMP%, %TEMP%, %USERPROFILE%, or the Windows directory.
	// On Plan 9, it returns /tmp.
	//
	// The directory is neither guaranteed to exist nor have accessible
	// permissions.
	GetTempDir() string

	// IsExist returns a boolean indicating whether the error is known to report
	// that a file or directory already exists. It is satisfied by ErrExist as
	// well as some syscall errors.
	IsExist(err error) bool

	// IsNotExist returns a boolean indicating whether the error is known to
	// report that a file or directory does not exist. It is satisfied by
	// ErrNotExist as well as some syscall errors.
	IsNotExist(err error) bool

	// Mkdir creates a new directory with the specified name and permission
	// bits (before umask).
	// If there is an error, it will be of type *PathError.
	Mkdir(name string, perm os.FileMode) error

	// MkdirAll creates a directory named path,
	// along with any necessary parents, and returns nil,
	// or else returns an error.
	// The permission bits perm (before umask) are used for all
	// directories that MkdirAll creates.
	// If path is already a directory, MkdirAll does nothing
	// and returns nil.
	MkdirAll(path string, perm os.FileMode) error

	// Open opens the named file for reading. If successful, methods on
	// the returned file can be used for reading; the associated file
	// descriptor has mode O_RDONLY.
	// If there is an error, it will be of type *PathError.
	Open(name string) (File, error)

	// OpenFile is the generalized open call; most users will use Open
	// or Create instead. It opens the named file with specified flag
	// (O_RDONLY etc.) and perm (before umask), if applicable. If successful,
	// methods on the returned File can be used for I/O.
	// If there is an error, it will be of type *PathError.
	OpenFile(name string, flag int, perm os.FileMode) (File, error)

	// ReadDir reads the directory named by dirname and returns
	// a list of directory entries sorted by filename.
	ReadDir(dirname string) ([]os.FileInfo, error)

	// ReadFile reads the file named by filename and returns the contents.
	// A successful call returns err == nil, not err == EOF. Because ReadFile
	// reads the whole file, it does not treat an EOF from Read as an error
	// to be reported.
	ReadFile(filename string) ([]byte, error)

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
	Rename(oldname, newname string) error

	// SameFile reports whether fi1 and fi2 describe the same file.
	// For example, on Unix this means that the device and inode fields
	// of the two underlying structures are identical; on other systems
	// the decision may be based on the path names.
	// SameFile only applies to results returned by this package's Stat.
	// It returns false in other cases.
	SameFile(fi1, fi2 os.FileInfo) bool

	// Stat returns a FileInfo describing the named file.
	// If there is an error, it will be of type *PathError.
	Stat(name string) (os.FileInfo, error)

	// TempDir creates a new temporary directory in the directory dir
	// with a name beginning with prefix and returns the path of the
	// new directory. If dir is the empty string, GetTempDir uses the
	// default directory for temporary files (see os.GetTempDir).
	// Multiple programs calling GetTempDir simultaneously
	// will not choose the same directory. It is the caller's responsibility
	// to remove the directory when no longer needed.
	TempDir(dir, prefix string) (name string, err error)

	// TempFile creates a new temporary file in the directory dir,
	// opens the file for reading and writing, and returns the resulting *os.File.
	// The filename is generated by taking pattern and adding a random
	// string to the end. If pattern includes a "*", the random string
	// replaces the last "*".
	// If dir is the empty string, TempFile uses the default directory
	// for temporary files (see os.TempDir).
	// Multiple programs calling TempFile simultaneously
	// will not choose the same file. The caller can use f.Type()
	// to find the pathname of the file. It is the caller's responsibility
	// to remove the file when no longer needed.
	TempFile(dir, pattern string) (f File, err error)

	// Truncate changes the size of the named file.
	// If the file is a symbolic link, it changes the size of the link's target.
	// If there is an error, it will be of type *PathError.
	Truncate(name string, size int64) error

	// WriteFile writes data to a file named by filename.
	// If the file does not exist, WriteFile creates it with permissions perm;
	// otherwise WriteFile truncates it before writing.
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

// ChDirer is the interface that wraps the Chdir and Getwd methods.
type ChDirer interface {
	// Chdir changes the current working directory to the named directory.
	// If there is an error, it will be of type *PathError.
	Chdir(dir string) error

	// Getwd returns a rooted path name corresponding to the
	// current directory. If the current directory can be
	// reached via multiple paths (due to symbolic links),
	// Getwd may return any one of them.
	Getwd() (dir string, err error)
}

// ChModer is the interface that wraps the Chmod method.
type ChModer interface {
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
	// On Windows, the mode must be non-zero but otherwise only the 0200
	// bit (owner writable) of mode is used; it controls whether the
	// file's read-only attribute is set or cleared. attribute. The other
	// bits are currently unused. Use mode 0400 for a read-only file and
	// 0600 for a readable+writable file.
	//
	// On Plan 9, the mode's permission bits, ModeAppend, ModeExclusive,
	// and ModeTemporary are used.
	Chmod(name string, mode os.FileMode) error
}

// ChOwner is the interface that wraps the Chown and Lchown methods.
type ChOwner interface {
	// Chown changes the numeric uid and gid of the named file.
	// If the file is a symbolic link, it changes the uid and gid of the link's target.
	// A uid or gid of -1 means to not change that value.
	// If there is an error, it will be of type *PathError.
	//
	// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
	// EPLAN9 error, wrapped in *PathError.
	Chown(name string, uid, gid int) error

	// Lchown changes the numeric uid and gid of the named file.
	// If the file is a symbolic link, it changes the uid and gid of the link itself.
	// If there is an error, it will be of type *PathError.
	//
	// On Windows, it always returns the syscall.EWINDOWS error, wrapped
	// in *PathError.
	Lchown(name string, uid, gid int) error
}

// ChRooter is the interface that wraps the Chroot method.
type ChRooter interface {
	// Chroot changes the root to that specified in path.
	// If the user has not root privileges avfs.errPermDenied is returned.
	// If there is an error, it will be of type *PathError.
	Chroot(path string) error
}

// ChTimer is the interface that wraps the Chtimes method.
type ChTimer interface {
	// Chtimes changes the access and modification times of the named
	// file, similar to the Unix utime() or utimes() functions.
	//
	// The underlying file system may truncate or round the values to a
	// less precise time unit.
	// If there is an error, it will be of type *PathError.
	Chtimes(name string, atime, mtime time.Time) error
}

// Cloner is the interface that wraps the Clone method.
type Cloner interface {
	// Clone returns a shallow copy of the current file system (see MemFs)
	// or the file system itself if does not support this feature (FeatClonable).
	Clone() Fs
}

// Featurer is the interface that wraps the Feature method.
type Featurer interface {
	// Features returns the set of features provided by the file system or identity manager.
	Features() Feature

	// HasFeature returns true if the file system or identity manager provides a given feature.
	HasFeature(feature Feature) bool
}

// HardLinker is the interface that wraps the Link method.
type HardLinker interface {
	// Link creates newname as a hard link to the oldname file.
	// If there is an error, it will be of type *LinkError.
	Link(oldname, newname string) error
}

// Namer is the the interface that wraps the name method.
type Namer interface {
	Name() string
}

// Pather is the interface that wraps all path related functions.
type Pather interface {
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

	// Dir returns all but the last element of path, typically the path's directory.
	// After dropping the final element, Dir calls Clean on the path and trailing
	// slashes are removed.
	// If the path is empty, Dir returns ".".
	// If the path consists entirely of separators, Dir returns a single separator.
	// The returned path does not end in a separator unless it is the root directory.
	Dir(path string) string

	// Glob returns the names of all files matching pattern or nil
	// if there is no matching file. The syntax of patterns is the same
	// as in Match. The pattern may describe hierarchical names such as
	// /usr/*/bin/ed (assuming the Separator is '/').
	//
	// Glob ignores file system errors such as I/O errors reading directories.
	// The only possible returned error is ErrBadPattern, when pattern
	// is malformed.
	Glob(pattern string) (matches []string, err error)

	// IsAbs reports whether the path is absolute.
	IsAbs(path string) bool

	// IsPathSeparator reports whether c is a directory separator character.
	IsPathSeparator(c uint8) bool

	// Join joins any number of path elements into a single path, adding a
	// separating slash if necessary. The result is Cleaned; in particular,
	// all empty strings are ignored.
	Join(elem ...string) string

	// Rel returns a relative path that is lexically equivalent to targpath when
	// joined to basepath with an intervening separator. That is,
	// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
	// On success, the returned path will always be relative to basepath,
	// even if basepath and targpath share no elements.
	// An error is returned if targpath can't be made relative to basepath or if
	// knowing the current working directory would be necessary to compute it.
	// Rel calls Clean on the result.
	Rel(basepath, targpath string) (string, error)

	// Split splits path immediately following the final Separator,
	// separating it into a directory and file name component.
	// If there is no Separator in path, Split returns an empty dir
	// and file set to path.
	// The returned values have the property that path = dir+file.
	Split(path string) (dir, file string)

	// Walk walks the file tree rooted at root, calling walkFn for each file or
	// directory in the tree, including root. All errors that arise visiting files
	// and directories are filtered by walkFn. The files are walked in lexical
	// order, which makes the output deterministic but means that for very
	// large directories Walk can be inefficient.
	// Walk does not follow symbolic links.
	Walk(root string, walkFn filepath.WalkFunc) error
}

// SymLinker is the interface that groups functions related to symbolic links.
type SymLinker interface {
	// EvalSymlinks returns the path name after the evaluation of any symbolic
	// links.
	// If path is relative the result will be relative to the current directory,
	// unless one of the components is an absolute symbolic link.
	// EvalSymlinks calls Clean on the result.
	EvalSymlinks(path string) (string, error)

	// Lstat returns a FileInfo describing the named file.
	// If the file is a symbolic link, the returned FileInfo
	// describes the symbolic link. Lstat makes no attempt to follow the link.
	// If there is an error, it will be of type *PathError.
	Lstat(name string) (os.FileInfo, error)

	// Readlink returns the destination of the named symbolic link.
	// If there is an error, it will be of type *PathError.
	Readlink(name string) (string, error)

	// Symlink creates newname as a symbolic link to oldname.
	// If there is an error, it will be of type *LinkError.
	Symlink(oldname, newname string) error
}

// Typer is the interface that wraps the Type method.
type Typer interface {
	// Type returns the type of the fileSystem or Identity manager.
	Type() string
}

// OSTyper is the interface that wraps the OSType method.
type OSTyper interface {
	// OSType returns the operating system type of the file system.
	OSType() OSType
}

// UMasker is the interface that groups functions related to file mode creation mask.
type UMasker interface {
	// GetUMask returns the file mode creation mask.
	GetUMask() os.FileMode

	// UMask sets the file mode creation mask.
	UMask(mask os.FileMode)
}

// File represents a file in the file system.
type File interface {
	BasicFile
	FileChDirer
	FileChModer
	FileChOwner
	FileSyncer
}

// BasicFile represents a basic file in the file system.
type BasicFile interface {
	io.Closer
	io.ReadWriteSeeker
	io.ReaderAt
	io.StringWriter
	io.WriterAt

	// Fd returns the integer Unix file descriptor referencing the open file.
	// The file descriptor is valid only until f.Close is called or f is garbage collected.
	// On Unix systems this will cause the SetDeadline methods to stop working.
	Fd() uintptr

	// Name returns the name of the file as presented to Open.
	Name() string

	// Readdir reads the contents of the directory associated with file and
	// returns a slice of up to n FileInfo values, as would be returned
	// by Lstat, in directory order. Subsequent calls on the same file will yield
	// further FileInfos.
	//
	// If n > 0, Readdir returns at most n FileInfo structures. In this case, if
	// Readdir returns an empty slice, it will return a non-nil error
	// explaining why. At the end of a directory, the error is io.EOF.
	//
	// If n <= 0, Readdir returns all the FileInfo from the directory in
	// a single slice. In this case, if Readdir succeeds (reads all
	// the way to the end of the directory), it returns the slice and a
	// nil error. If it encounters an error before the end of the
	// directory, Readdir returns the FileInfo read until that point
	// and a non-nil error.
	Readdir(n int) ([]os.FileInfo, error)

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

	// Stat returns the FileInfo structure describing file.
	// If there is an error, it will be of type *PathError.
	Stat() (os.FileInfo, error)

	// Truncate changes the size of the file.
	// It does not change the I/O offset.
	// If there is an error, it will be of type *PathError.
	Truncate(size int64) error
}

// FileChDirer is the interface that wraps the Chdir method of a File.
type FileChDirer interface {
	// Chdir changes the current working directory to the file,
	// which must be a directory.
	// If there is an error, it will be of type *PathError.
	Chdir() error
}

// FileChModer is the interface that wraps the Chmod method of a File.
type FileChModer interface {
	// Chmod changes the mode of the file to mode.
	// If there is an error, it will be of type *PathError.
	Chmod(mode os.FileMode) error
}

// FileChOwner is the interface that wraps the Chown method of a File.
type FileChOwner interface {
	// Chown changes the numeric uid and gid of the named file.
	// If there is an error, it will be of type *PathError.
	//
	// On Windows, it always returns the syscall.EWINDOWS error, wrapped
	// in *PathError.
	Chown(uid, gid int) error
}

// FileSyncer is the interface that wraps the Sync method of a file.
type FileSyncer interface {
	// Sync commits the current contents of the file to stable storage.
	// Typically, this means flushing the file system's in-memory copy
	// of recently written data to disk.
	Sync() error
}

// IdentityMgr interface manages identities (users and groups).
type IdentityMgr interface {
	Featurer
	GroupMgr
	UserMgr
	Typer
}

// GroupMgr interface manages groups.
type GroupMgr interface {
	// GroupAdd adds a new group.
	// If the group already exists, the returned error is of type AlreadyExistsGroupError.
	GroupAdd(name string) (GroupReader, error)

	// GroupDel deletes an existing group.
	// If the group cannot be found, the returned error is of type UnknownGroupError.
	GroupDel(name string) error

	// LookupGroup looks up a group by name.
	// If the group cannot be found, the returned error is of type UnknownGroupError.
	LookupGroup(name string) (GroupReader, error)

	// LookupGroupId looks up a group by groupid.
	// If the group cannot be found, the returned error is of type UnknownGroupIdError.
	LookupGroupId(gid int) (GroupReader, error)
}

// GroupReader interface reads group information.
type GroupReader interface {
	// Gid returns the group id.
	Gid() int

	// Name returns the group name.
	Name() string
}

// UserMgr interface manages the users.
type UserMgr interface {
	// UserAdd adds a new user.
	// If the user already exists, the returned error is of type AlreadyExistsUserError.
	UserAdd(name, groupName string) (UserReader, error)

	// UserDel deletes an existing user.
	UserDel(name string) error

	// LookupUser looks up a user by username.
	// If the user cannot be found, the returned error is of type UnknownUserError.
	LookupUser(name string) (UserReader, error)

	// LookupUserId looks up a user by userid.
	// If the user cannot be found, the returned error is of type UnknownUserIdError.
	LookupUserId(uid int) (UserReader, error)
}

// UserConnecter interface manages user connections.
type UserConnecter interface {
	// CurrentUser returns the current user.
	CurrentUser() UserReader

	// User sets and returns the current user.
	// If the user cannot be found, the returned error is of type UnknownUserError.
	User(name string) (UserReader, error)
}

// UserReader reads user information.
type UserReader interface {
	// Gid returns the primary group id.
	Gid() int

	// IsRoot returns true if the user has root privileges.
	IsRoot() bool

	// Name returns the user name.
	Name() string

	// Uid returns the user id.
	Uid() int
}

// StatT is the structure returned by os.FileInfo.Sys() for file systems other than OsFs.
// OsFs returns a syscall.Stat_t type on linux.
type StatT struct {
	Uid uint32
	Gid uint32
}
