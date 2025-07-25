//
//  Copyright 2021 The AVFS authors
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
	"io/fs"
	"reflect"
	"strconv"
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

// InvalidNameError is return when a username or a group name is invalid.
type InvalidNameError string

func (e InvalidNameError) Error() string { return "name: invalid name " + string(e) }

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

// ErrorIdentifier is the interface that wraps the Is method of an error.
type ErrorIdentifier interface {
	error

	// Is returns true if the error can be treated as equivalent to a target error.
	// target is one of fs.ErrPermission, fs.ErrExist, fs.ErrNotExist.
	Is(target error) bool
}

const customErrorBase = 2 << 30

type CustomError uintptr

//go:generate stringer -type CustomError -linecomment -output errors_custom.go

const (
	ErrNegativeOffset      CustomError = customErrorBase + 1 // negative offset
	ErrFileClosing         CustomError = customErrorBase + 2 // use of closed file
	ErrPatternHasSeparator CustomError = customErrorBase + 3 // pattern contains path separator
	ErrVolumeAlreadyExists CustomError = customErrorBase + 4 // Volume already exists.
	ErrVolumeNameInvalid   CustomError = customErrorBase + 5 // Volume name is invalid.
	ErrVolumeWindows       CustomError = customErrorBase + 6 // Volumes are available for Windows only.
)

func (i CustomError) Error() string {
	return i.String()
}

// LinuxError replaces syscall.Errno for Linux operating systems.
type LinuxError uintptr

//go:generate stringer -type LinuxError -linecomment -output errors_forlinux.go

// Errors for Linux operating systems.
// See https://github.com/torvalds/linux/blob/master/tools/include/uapi/asm-generic/errno-base.h
const (
	ErrBadFileDesc     LinuxError = errEBADF     // bad file descriptor
	ErrCrossDevLink    LinuxError = errEXDEV     // invalid cross-device link
	ErrDirNotEmpty     LinuxError = errENOTEMPTY // directory not empty
	ErrFileExists      LinuxError = errEEXIST    // file exists
	ErrInvalidArgument LinuxError = errEINVAL    // invalid argument
	ErrIsADirectory    LinuxError = errEISDIR    // is a directory
	ErrNoSuchFileOrDir LinuxError = errENOENT    // no such file or directory
	ErrNotADirectory   LinuxError = errENOTDIR   // not a directory
	ErrOpNotPermitted  LinuxError = errEPERM     // operation not permitted
	ErrPermDenied      LinuxError = errEACCES    // permission denied
	ErrTooManySymlinks LinuxError = errELOOP     // too many levels of symbolic links

	errEACCES    = 0xd
	errEBADF     = 0x9
	errEEXIST    = 0x11
	errEINVAL    = 0x16
	errEISDIR    = 0x15
	errENOENT    = 0x2
	errELOOP     = 0x28
	errENOTDIR   = 0x14
	errENOTEMPTY = 0x27
	errEPERM     = 0x1
	errEXDEV     = 0x12
)

// Error returns the error string of the Linux operating system.
func (i LinuxError) Error() string {
	return i.String()
}

// Is returns true if the LinuxError can be treated as equivalent to a target error.
// target is one of fs.ErrPermission, fs.ErrExist, fs.ErrNotExist.
func (i LinuxError) Is(target error) bool {
	switch target {
	case fs.ErrPermission:
		return i == ErrPermDenied || i == ErrOpNotPermitted
	case fs.ErrExist:
		return i == ErrFileExists || i == ErrDirNotEmpty
	case fs.ErrNotExist:
		return i == ErrNoSuchFileOrDir
	}

	return false
}

// WindowsError replaces syscall.Errno for Windows operating systems.
type WindowsError uintptr

//go:generate stringer -type WindowsError -linecomment -output errors_forwindows.go

// Errors for Windows operating systems.
// See https://learn.microsoft.com/en-us/windows/win32/debug/system-error-codes
const (
	ErrWinAccessDenied     WindowsError = 5          // Access is denied.
	ErrWinAlreadyExists    WindowsError = 183        // Cannot create a file when that file already exists.
	ErrWinBadNetPath       WindowsError = 53         // Bad network path.
	ErrWinDirNameInvalid   WindowsError = 0x10B      // The directory name is invalid.
	ErrWinDirNotEmpty      WindowsError = 145        // The directory is not empty.
	ErrWinFileExists       WindowsError = 80         // The file exists.
	ErrWinFileNotFound     WindowsError = 2          // The system cannot find the file specified.
	ErrWinIncorrectFunc    WindowsError = 1          // Incorrect function.
	ErrWinInvalidParameter WindowsError = 0x57       // The parameter is incorrect.
	ErrWinIsADirectory     WindowsError = 0x2000002A // is a directory
	ErrWinNegativeSeek     WindowsError = 0x83       // An attempt was made to move the file pointer before the beginning of the file.
	ErrWinNotReparsePoint  WindowsError = 4390       // The file or directory is not a reparse point.
	ErrWinInvalidHandle    WindowsError = 6          // The handle is invalid.
	ErrWinSharingViolation WindowsError = 32         // The process cannot access the file because it is being used by another process.
	ErrWinNotSupported     WindowsError = 0x20000082 // not supported by windows
	ErrWinPathNotFound     WindowsError = 3          // The system cannot find the path specified.
	ErrWinPrivilegeNotHeld WindowsError = 1314       // A required privilege is not held by the client.
)

// Error returns the error string of the Windows operating system.
func (i WindowsError) Error() string {
	return i.String()
}

// Is returns true if the WindowsError can be treated as equivalent to a target error.
// target is one of fs.ErrPermission, fs.ErrExist, fs.ErrNotExist.
func (i WindowsError) Is(target error) bool {
	switch target {
	case fs.ErrPermission:
		return i == ErrWinAccessDenied
	case fs.ErrExist:
		return i == ErrWinAlreadyExists || i == ErrWinDirNotEmpty || i == ErrWinFileExists
	case fs.ErrNotExist:
		return i == ErrWinFileNotFound || i == ErrWinBadNetPath || i == ErrWinPathNotFound
	}

	return false
}

// ErrorsForOS regroups errors depending on the OS emulated.
type ErrorsForOS struct {
	BadFileDesc     error // bad file descriptor.
	DirNotEmpty     error // Directory not empty.
	FileClosing     error // use of closed file.
	FileExists      error // File exists.
	InvalidArgument error // invalid argument.
	IsADirectory    error // File Is a directory.
	NegativeSeek    error // Negative seek invalid argument.
	NoSuchDir       error // No such directory.
	NoSuchFile      error // No such file.
	NotADirectory   error // Not a directory.
	OpNotPermitted  error // operation not permitted.
	PermDenied      error // Permission denied.
	TooManySymlinks error // Too many levels of symbolic links.
}

// errorsForOS defines common errors for each OS.
var errorsForOS = map[OSType]*ErrorsForOS{ //nolint:gochecknoglobals // Should be a const.
	OsLinux: {
		BadFileDesc:     ErrBadFileDesc,
		DirNotEmpty:     ErrDirNotEmpty,
		FileClosing:     ErrFileClosing,
		FileExists:      ErrFileExists,
		InvalidArgument: ErrInvalidArgument,
		IsADirectory:    ErrIsADirectory,
		NegativeSeek:    ErrInvalidArgument,
		NoSuchDir:       ErrNoSuchFileOrDir,
		NoSuchFile:      ErrNoSuchFileOrDir,
		NotADirectory:   ErrNotADirectory,
		OpNotPermitted:  ErrOpNotPermitted,
		PermDenied:      ErrPermDenied,
		TooManySymlinks: ErrTooManySymlinks,
	},
	OsDarwin: {
		BadFileDesc:     ErrBadFileDesc,
		DirNotEmpty:     ErrDirNotEmpty,
		FileClosing:     ErrFileClosing,
		FileExists:      ErrFileExists,
		InvalidArgument: ErrInvalidArgument,
		IsADirectory:    ErrIsADirectory,
		NegativeSeek:    ErrInvalidArgument,
		NoSuchDir:       ErrNoSuchFileOrDir,
		NoSuchFile:      ErrNoSuchFileOrDir,
		NotADirectory:   ErrNotADirectory,
		OpNotPermitted:  ErrOpNotPermitted,
		PermDenied:      ErrPermDenied,
		TooManySymlinks: ErrTooManySymlinks,
	},
	OsWindows: {
		BadFileDesc:     ErrWinAccessDenied,
		DirNotEmpty:     ErrWinDirNotEmpty,
		FileClosing:     ErrWinInvalidHandle,
		FileExists:      ErrWinFileExists,
		InvalidArgument: ErrWinInvalidParameter,
		IsADirectory:    ErrWinIsADirectory,
		NegativeSeek:    ErrWinNegativeSeek,
		NoSuchDir:       ErrWinPathNotFound,
		NoSuchFile:      ErrWinFileNotFound,
		NotADirectory:   ErrWinPathNotFound,
		OpNotPermitted:  ErrWinNotSupported,
		PermDenied:      ErrWinAccessDenied,
		TooManySymlinks: ErrTooManySymlinks,
	},
}

func ErrorsFor(osType OSType) *ErrorsForOS {
	errors, ok := errorsForOS[osType]
	if !ok {
		panic("Unknown OS type : " + osType.String())
	}

	return errors
}

// Operation constants (most used).
const (
	OpReaddirent = "readdirent"
	OpReaddir    = "readdir"
)
