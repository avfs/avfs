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

package basepathfs

import (
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/avfs/avfs"
)

// New returns a new base path file system (BasePathFS).
func New(baseFS avfs.VFS, basePath string) *BasePathFS {
	vfs, err := NewWithErr(baseFS, basePath)
	if err != nil {
		panic(err)
	}

	return vfs
}

// NewWithErr returns a new base path file system (BasePathFS).
func NewWithErr(baseFS avfs.VFS, basePath string) (*BasePathFS, error) {
	const op = "basepath"

	absPath, err := baseFS.Abs(basePath)
	if err != nil {
		return nil, err
	}

	info, err := baseFS.Stat(absPath)
	if err != nil {
		err = &fs.PathError{Op: op, Path: basePath, Err: errors.Unwrap(err)}

		return nil, err
	}

	if !info.IsDir() {
		err = &fs.PathError{Op: op, Path: basePath, Err: avfs.ErrNotADirectory}

		return nil, err
	}

	vfs := &BasePathFS{
		baseFS:   baseFS,
		basePath: absPath,
	}

	_ = vfs.SetFeatures(baseFS.Features() &^ avfs.FeatSymlink)
	vfs.err = avfs.ErrorsFor(vfs.OSType())

	return vfs, nil
}

// FromBasePath returns a BasePathFS path from an internal path.
// When the base path is "/base/path", FromBasePath("/base/path/tmp") returns "/tmp".
func (vfs *BasePathFS) FromBasePath(path string) string {
	if path == "" {
		return ""
	}

	if !strings.HasPrefix(path, vfs.basePath) {
		panic("path must start with " + vfs.basePath + " : " + path)
	}

	vl := avfs.VolumeNameLen(vfs, path)

	return vfs.Join(path[:vl], path[len(vfs.basePath):], string(vfs.PathSeparator()))
}

// FromPathError restore paths in fs.PathError if necessary.
func (vfs *BasePathFS) FromPathError(err error) error {
	e, ok := err.(*fs.PathError)
	if !ok {
		return err
	}

	return &fs.PathError{Op: e.Op, Path: vfs.FromBasePath(e.Path), Err: e.Err}
}

// FromLinkError restore paths in os.LinkError if necessary.
func (vfs *BasePathFS) FromLinkError(err error) error {
	e, ok := err.(*os.LinkError)
	if !ok {
		return err
	}

	return &os.LinkError{Op: e.Op, Old: vfs.FromBasePath(e.Old), New: vfs.FromBasePath(e.New), Err: e.Err}
}

// ToBasePath transforms a BasePathFS path to an internal path.
// When the base path is "/base/path", ToBasePath("/tmp") returns "/base/path/tmp".
func (vfs *BasePathFS) ToBasePath(path string) string {
	if path == "" {
		return ""
	}

	if path == "/" {
		return vfs.basePath
	}

	if vfs.IsAbs(path) {
		vl := avfs.VolumeNameLen(vfs, path)

		return vfs.Join(vfs.basePath, path[vl:])
	}

	return path
}

// Name returns the name of the fileSystem.
func (vfs *BasePathFS) Name() string {
	return vfs.baseFS.Name()
}

// OSType returns the operating system type of the file system.
func (vfs *BasePathFS) OSType() avfs.OSType {
	return vfs.baseFS.OSType()
}

// Type returns the type of the fileSystem or Identity manager.
func (*BasePathFS) Type() string {
	return "BasePathFS"
}
