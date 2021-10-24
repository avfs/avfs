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

// Errno replaces syscall.Errno for all OSes.
type Errno uint64 //nolint:errname // the type name `Errno` should conform to the `XxxError` format.

func (en Errno) Error() string {
	i := en + Errno(Cfg.OSType())<<32

	s, ok := errText[i]
	if ok {
		return s
	}

	return "errno " + strconv.Itoa(int(en))
}

const (
	// Errors on Linux operating systems.
	// Most of the errors below can be found there :
	// https://github.com/torvalds/linux/blob/master/tools/include/uapi/asm-generic/errno-base.h

	ErrBadFileDesc     = errEBADF     // bad file descriptor.
	ErrDirNotEmpty     = errENOTEMPTY // Directory not empty.
	ErrFileExists      = errEEXIST    // File exists.
	ErrInvalidArgument = errEINVAL    // invalid argument
	ErrIsADirectory    = errEISDIR    // File Is a directory.
	ErrNoSuchFileOrDir = errENOENT    // No such file or directory.
	ErrNotADirectory   = errENOTDIR   // Not a directory.
	ErrOpNotPermitted  = errEPERM     // operation not permitted.
	ErrPermDenied      = errEACCES    // Permission denied.
	ErrTooManySymlinks = errELOOP     // Too many levels of symbolic links.

	errEACCES    = Errno(0xd)
	errEBADF     = Errno(0x9)
	errEEXIST    = Errno(0x11)
	errEINVAL    = Errno(0x16)
	errEISDIR    = Errno(0x15)
	errENOENT    = Errno(0x2)
	errELOOP     = Errno(0x28)
	errENOTDIR   = Errno(0x14)
	errENOTEMPTY = Errno(0x27)
	errEPERM     = Errno(0x1)

	// Errors on Windows operating systems.

	ErrWinAccessDenied     = Errno(5)          // Access is denied.
	ErrWinDirNameInvalid   = Errno(0x10B)      // The directory name is invalid.
	ErrWinDirNotEmpty      = Errno(145)        // The directory is not empty.
	ErrWinFileExists       = Errno(80)         // The file exists.
	ErrWinFileNotFound     = Errno(2)          // The system cannot find the file specified.
	ErrWinIsADirectory     = Errno(21)         // is a directory
	ErrWinNegativeSeek     = Errno(0x83)       // An attempt was made to move the file pointer before the beginning of the file.
	ErrWinNotReparsePoint  = Errno(4390)       // The file or directory is not a reparse point.
	ErrWinInvalidHandle    = Errno(6)          // The handle is invalid.
	ErrWinNotSupported     = Errno(0x20000082) // not supported by windows
	ErrWinPathNotFound     = Errno(3)          // The system cannot find the path specified.
	ErrWinPrivilegeNotHeld = Errno(1314)       // A required privilege is not held by the client.

	linuxError   = Errno(OsLinux) << 32
	windowsError = Errno(OsWindows) << 32
)

// errText translates an OS error number to text for all OSes.
var errText = map[Errno]string{
	ErrBadFileDesc + linuxError:     "bad file descriptor",
	ErrDirNotEmpty + linuxError:     "directory not empty",
	ErrFileExists + linuxError:      "file exists",
	ErrInvalidArgument + linuxError: "invalid argument",
	ErrIsADirectory + linuxError:    "is a directory",
	ErrNoSuchFileOrDir + linuxError: "no such file or directory",
	ErrNotADirectory + linuxError:   "not a directory",
	ErrOpNotPermitted + linuxError:  "operation not permitted",
	ErrPermDenied + linuxError:      "permission denied",
	ErrTooManySymlinks + linuxError: "too many levels of symbolic links",

	ErrWinAccessDenied + windowsError:     "Access is denied.",
	ErrWinDirNameInvalid + windowsError:   "The directory name is invalid.",
	ErrWinDirNotEmpty + windowsError:      "The directory is not empty.",
	ErrWinFileExists + windowsError:       "The file exists.",
	ErrWinFileNotFound + windowsError:     "The system cannot find the file specified.",
	ErrWinIsADirectory + windowsError:     "is a directory",
	ErrWinNegativeSeek + windowsError:     "An attempt was made to move the file pointer before the beginning of the file.",
	ErrWinNotReparsePoint + windowsError:  "The file or directory is not a reparse point.",
	ErrWinInvalidHandle + windowsError:    "The handle is invalid.",
	ErrWinNotSupported + windowsError:     "not supported by windows",
	ErrWinPathNotFound + windowsError:     "The system cannot find the path specified.",
	ErrWinPrivilegeNotHeld + windowsError: "A required privilege is not held by the client.",
}
