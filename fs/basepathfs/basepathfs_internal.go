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
	"os"
	"strings"
)

// pathFsToBpFs transforms the absolute path of the base file system from the BasePathFS path.
func (vfs *BasePathFS) pathFsToBpFs(path string) string {
	if path == "" {
		return ""
	}

	absPath, _ := vfs.Abs(path)

	return vfs.basePath + absPath
}

// pathBpFsToFs transforms the absolute path of the BasePathFS path to the base file system path.
func (vfs *BasePathFS) pathBpFsToFs(path string) string {
	return strings.TrimPrefix(path, vfs.basePath)
}

// restoreError restore paths in errors if necessary.
func (vfs *BasePathFS) restoreError(err error) error {
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *os.PathError:
		return &os.PathError{
			Op:   e.Op,
			Path: vfs.pathBpFsToFs(e.Path),
			Err:  e.Err,
		}
	case *os.LinkError:
		return &os.LinkError{
			Op:  e.Op,
			Old: vfs.pathBpFsToFs(e.Old),
			New: vfs.pathBpFsToFs(e.New),
			Err: e.Err,
		}
	case *os.SyscallError:
		return &os.SyscallError{
			Syscall: strings.ReplaceAll(e.Syscall, vfs.basePath, ""),
			Err:     e.Err,
		}
	default:
		return err
	}
}
