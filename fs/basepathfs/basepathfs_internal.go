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

// pathFsToBpFs transforms the absolute path of the base file system from the BasePathFs path.
func (fs *BasePathFs) pathFsToBpFs(path string) string {
	if path == "" {
		return ""
	}

	absPath, _ := fs.Abs(path)

	return fs.basePath + absPath
}

// pathBpFsToFs transforms the absolute path of the BasePathFs path to the base file system path.
func (fs *BasePathFs) pathBpFsToFs(path string) string {
	return strings.TrimPrefix(path, fs.basePath)
}

// restoreError restore paths in errors if necessary.
func (fs *BasePathFs) restoreError(err error) error {
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *os.PathError:
		return &os.PathError{
			Op:   e.Op,
			Path: fs.pathBpFsToFs(e.Path),
			Err:  e.Err,
		}
	case *os.LinkError:
		return &os.LinkError{
			Op:  e.Op,
			Old: fs.pathBpFsToFs(e.Old),
			New: fs.pathBpFsToFs(e.New),
			Err: e.Err,
		}
	case *os.SyscallError:
		return &os.SyscallError{
			Syscall: strings.ReplaceAll(e.Syscall, fs.basePath, ""),
			Err:     e.Err,
		}
	default:
		return err
	}
}
