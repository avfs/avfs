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

// toBasePath returns the real absolute path of the base file system from the BasePathFs path.
func (fs *BasePathFs) toBasePath(path string) string {
	if path == "" {
		return ""
	}

	absPath, _ := fs.Abs(path)

	return fs.basePath + absPath
}

// restoreError paths in errors if necessary.
func (fs *BasePathFs) restoreError(err error) error {
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *os.PathError:
		return &os.PathError{
			Op:   e.Op,
			Path: strings.TrimPrefix(e.Path, fs.basePath),
			Err:  e.Err,
		}
	case *os.LinkError:
		return &os.LinkError{
			Op:  e.Op,
			Old: strings.TrimPrefix(e.Old, fs.basePath),
			New: strings.TrimPrefix(e.New, fs.basePath),
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
