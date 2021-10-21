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
	"errors"
	"reflect"
	"strconv"
)

var (
	// ErrNegativeOffset is the Error negative offset.
	ErrNegativeOffset = errors.New("negative offset")

	// ErrFileClosing is returned when a file descriptor is used after it has been closed.
	ErrFileClosing = errors.New("use of closed file")

	// ErrPatternHasSeparator is returned when a bad pattern is used in CreateTemp or MkdirTemp.
	ErrPatternHasSeparator = errors.New("pattern contains path separator")
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

// Error replaces syscall.Errno
type Error uint64

func (e Error) Error() string {
	s, ok := Errors[e]
	if ok {
		return s
	}

	return ""
}

const (
	// Errors on linux operating systems.
	// Most of the errors below can be found there :
	// https://github.com/torvalds/linux/blob/master/tools/include/uapi/asm-generic/errno-base.h
	linuxError = uint64(OsLinux) >> 32

	ErrBadFileDesc     = Error(linuxError + 0x9)  // bad file descriptor.
	ErrDirNotEmpty     = Error(linuxError + 0x27) // Directory not empty.
	ErrFileExists      = Error(linuxError + 0x11) // File exists.
	ErrInvalidArgument = Error(linuxError + 0x16) // invalid argument
	ErrIsADirectory    = Error(linuxError + 0x15) // File Is a directory.
	ErrNoSuchFileOrDir = Error(linuxError + 0x2)  // No such file or directory.
	ErrNotADirectory   = Error(linuxError + 0x14) // Not a directory.
	ErrOpNotPermitted  = Error(linuxError + 0x1)  // operation not permitted.
	ErrPermDenied      = Error(linuxError + 0xd)  // Permission denied.
	ErrTooManySymlinks = Error(linuxError + 0x28) // Too many levels of symbolic links.

	// Errors on Windows operating systems only.
	windowsError = uint64(OsWindows) >> 32

	ErrWinAccessDenied     = Error(windowsError + 0x5)        // Access is denied.
	ErrWinDirNameInvalid   = Error(windowsError + 0x10B)      // The directory name is invalid.
	ErrWinDirNotEmpty      = Error(windowsError + 145)        // The directory is not empty.
	ErrWinFileExists       = Error(windowsError + 80)         // The file exists.
	ErrWinNegativeSeek     = Error(windowsError + 0x83)       // An attempt was made to move the file pointer before the beginning of the file.
	ErrWinNotReparsePoint  = Error(windowsError + 4390)       // The file or directory is not a reparse point.
	ErrWinInvalidHandle    = Error(windowsError + 0x6)        // The handle is invalid.
	ErrWinNotSupported     = Error(windowsError + 0x20000082) // Not supported by windows.
	ErrWinPathNotFound     = Error(windowsError + 0x3)        // The system cannot find the path specified.
	ErrWinPrivilegeNotHeld = Error(windowsError + 1314)       // A required privilege is not held by the client.
)

type ErrorText map[Error]string

var Errors = ErrorText{
	ErrBadFileDesc:         "bad file descriptor",
	ErrDirNotEmpty:         "directory not empty",
	ErrFileExists:          "file exists",
	ErrInvalidArgument:     "invalid argument",
	ErrIsADirectory:        "is a directory",
	ErrNoSuchFileOrDir:     "no such file or directory",
	ErrNotADirectory:       "not a directory",
	ErrOpNotPermitted:      "operation not permitted",
	ErrPermDenied:          "permission denied",
	ErrTooManySymlinks:     "too many levels of symbolic links",
	ErrWinAccessDenied:     "Access is denied.",
	ErrWinDirNameInvalid:   "The directory name is invalid.",
	ErrWinDirNotEmpty:      "The directory is not empty.",
	ErrWinFileExists:       "The file exists.",
	ErrWinNegativeSeek:     "An attempt was made to move the file pointer before the beginning of the file.",
	ErrWinNotReparsePoint:  "The file or directory is not a reparse point.",
	ErrWinInvalidHandle:    "The handle is invalid.",
	ErrWinNotSupported:     "Not supported by windows.",
	ErrWinPathNotFound:     "The system cannot find the path specified.",
	ErrWinPrivilegeNotHeld: "A required privilege is not held by the client.",
}
