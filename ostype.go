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
	"errors"
	"runtime"
)

var ErrSetOSType = errors.New("can't set OS type, use build tag 'avfs_setostype' to set OS type")

// OSType defines the operating system type.
type OSType uint16

//go:generate stringer -type OSType -linecomment -output ostype_string.go

const (
	OsUnknown OSType = iota // Unknown
	OsLinux                 // Linux
	OsWindows               // Windows
	OsDarwin                // Darwin
)

// CurrentOSType returns the current OSType.
func CurrentOSType() OSType {
	return currentOSType
}

// currentOSType is the current OSType.
var currentOSType = func() OSType { //nolint:gochecknoglobals // Store the current OS Type.
	switch runtime.GOOS {
	case "linux":
		return OsLinux
	case "darwin":
		return OsDarwin
	case "windows":
		return OsWindows
	default:
		return OsUnknown
	}
}()

// OSTyper is the interface that wraps the OS type related methods.
type OSTyper interface {
	// OSType returns the operating system type of the file system.
	OSType() OSType
}

// OSTypeFn provides OS type functions to a file system or an identity manager.
type OSTypeFn struct {
	osType        OSType // OSType defines the operating system type.
	pathSeparator uint8  // pathSeparator is the OS-specific path separator.
}

// OSType returns the operating system type of the file system.
func (osf *OSTypeFn) OSType() OSType {
	return osf.osType
}

// PathSeparator return the OS-specific path separator.
func (osf *OSTypeFn) PathSeparator() uint8 {
	return osf.pathSeparator
}

// SetOSType sets the operating system Type.
// If the OS type can't be changed it returns an error.
func (osf *OSTypeFn) SetOSType(osType OSType) error {
	if osType == OsUnknown {
		osType = CurrentOSType()
	}

	if BuildFeatures()&FeatSetOSType != 0 && osType != CurrentOSType() {
		return ErrSetOSType
	}

	osf.osType = osType

	sep := uint8('/')
	if osType == OsWindows {
		sep = '\\'
	}

	osf.pathSeparator = sep

	return nil
}
