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

package basepathfs

import (
	"github.com/avfs/avfs"
)

// CloneWithUser returns a shallow copy of the current file system (see
// BasePathFS) acting as user and emulating ost.
//
// The base file system is cloned, and the copy is scoped to the same base path.
// If the base file system is not clonable, the returned error is of type
// avfs.ErrNotSupported.
func (vfs *BasePathFS) CloneWithUser(user avfs.UserReader, ost avfs.OSType) (avfs.VFS, error) {
	cloner, ok := vfs.baseFS.(avfs.Cloner)
	if !ok {
		return nil, avfs.ErrNotSupported
	}

	baseFS, err := cloner.CloneWithUser(user, ost)
	if err != nil {
		return nil, err
	}

	return NewWithErr(baseFS, vfs.basePath)
}

// CloneWithUserName returns a shallow copy of the current file system (see
// BasePathFS) acting as the user userName and emulating ost.
//
// If the user is not found, the returned error is of type
// avfs.UnknownUserError. If the base file system is not clonable, the returned
// error is of type avfs.ErrNotSupported.
func (vfs *BasePathFS) CloneWithUserName(userName string, ost avfs.OSType) (avfs.VFS, error) {
	cloner, ok := vfs.baseFS.(avfs.Cloner)
	if !ok {
		return nil, avfs.ErrNotSupported
	}

	baseFS, err := cloner.CloneWithUserName(userName, ost)
	if err != nil {
		return nil, err
	}

	return NewWithErr(baseFS, vfs.basePath)
}
