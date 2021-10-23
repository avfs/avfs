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
	s, ok := errText[en]
	if ok {
		return s
	}

	return "errno " + strconv.Itoa(int(en))
}

type errnoOSType struct {
	En  Errno
	Ost OSType
}

// errTransMap stores translations of Linux errors to other operating system errors.
var errTransMap = map[errnoOSType]Errno{
	{En: ErrNoSuchFileOrDir, Ost: OsWindows}: ErrWinPathNotFound,
	{En: ErrFileExists, Ost: OsWindows}:      ErrWinFileExists,
}

// ErrTranslate translates Linux errors to other operating systems errors.
func ErrTranslate(err error, ost OSType) error {
	if ost == OsLinux {
		return err
	}

	en, ok := err.(Errno)
	if !ok {
		return err
	}

	et, ok := errTransMap[errnoOSType{En: en, Ost: ost}]
	if ok {
		return et
	}

	return err
}

type opOSType struct {
	Op  string
	Ost OSType
}

// opTransMap stores translations of Linux operations to other operating system operations.
var opTransMap = map[opOSType]string{ //nolint:gochecknoglobals // opTransMap is a global variable
	{Op: "lstat", Ost: OsWindows}: "CreateFile",
	{Op: "stat", Ost: OsWindows}:  "CreateFile",
}

// OpTranslate translates linux operation strings to other OS operation strings.
func OpTranslate(op string, ost OSType) string {
	if ost == OsLinux {
		return op
	}

	opt, ok := opTransMap[opOSType{Op: op, Ost: ost}]
	if ok {
		return opt
	}

	return op
}

const (
	// Errors on Linux operating systems.
	// Most of the errors below can be found there :
	// https://github.com/torvalds/linux/blob/master/tools/include/uapi/asm-generic/errno-base.h
	linuxError = Errno(OsLinux) << 32

	ErrBadFileDesc     = linuxError + errEBADF     // bad file descriptor.
	ErrDirNotEmpty     = linuxError + errENOTEMPTY // Directory not empty.
	ErrFileExists      = linuxError + errEEXIST    // File exists.
	ErrInvalidArgument = linuxError + errEINVAL    // invalid argument
	ErrIsADirectory    = linuxError + errEISDIR    // File Is a directory.
	ErrNoSuchFileOrDir = linuxError + errENOENT    // No such file or directory.
	ErrNotADirectory   = linuxError + errENOTDIR   // Not a directory.
	ErrOpNotPermitted  = linuxError + errEPERM     // operation not permitted.
	ErrPermDenied      = linuxError + errEACCES    // Permission denied.
	ErrTooManySymlinks = linuxError + errELOOP     // Too many levels of symbolic links.

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
	windowsError = Errno(OsWindows) << 32

	ErrWinAccessDenied     = windowsError + 0x5        // Access is denied.
	ErrWinDirNameInvalid   = windowsError + 0x10B      // The directory name is invalid.
	ErrWinDirNotEmpty      = windowsError + 145        // The directory is not empty.
	ErrWinFileExists       = windowsError + 80         // The file exists.
	ErrWinNegativeSeek     = windowsError + 0x83       // An attempt was made to move the file pointer before the beginning of the file.
	ErrWinNotReparsePoint  = windowsError + 4390       // The file or directory is not a reparse point.
	ErrWinInvalidHandle    = windowsError + 0x6        // The handle is invalid.
	ErrWinNotSupported     = windowsError + 0x20000082 // Not supported by windows.
	ErrWinPathNotFound     = windowsError + 0x3        // The system cannot find the path specified.
	ErrWinPrivilegeNotHeld = windowsError + 1314       // A required privilege is not held by the client.
)

// errText translates an OS error number to text for all OSes.
var errText = map[Errno]string{
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
