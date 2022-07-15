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
	"io/fs"
	"os"
	"strings"
)

// toBasePath transforms a BasePathFS path to a BaseFS path.
func (vfs *BasePathFS) toBasePath(path string) string {
	if vfs.IsAbs(path) {
		vl := vfs.Utils.VolumeNameLen(path)

		return vfs.basePath + path[vl:]
	}

	return path
}

// fromBasePath returns a BasePathFS path from a BaseFs path.
func (vfs *BasePathFS) fromBasePath(path string) string {
	if strings.HasPrefix(path, vfs.basePath) {
		vl := vfs.Utils.VolumeNameLen(path)

		return path[:vl] + path[len(vfs.basePath):]
	}

	return path
}

// restoreError restore paths in errors if necessary.
func (vfs *BasePathFS) restoreError(err error) error {
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *fs.PathError:
		return &fs.PathError{
			Op:   e.Op,
			Path: vfs.fromBasePath(e.Path),
			Err:  e.Err,
		}
	case *os.LinkError:
		return &os.LinkError{
			Op:  e.Op,
			Old: vfs.fromBasePath(e.Old),
			New: vfs.fromBasePath(e.New),
			Err: e.Err,
		}
	default:
		return err
	}
}
