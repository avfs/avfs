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
	"os"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
)

// New returns a new base path file system (BasePathFs).
func New(baseFs avfs.Fs, basePath string) (*BasePathFs, error) {
	const op = "basepath"

	absPath, _ := baseFs.Abs(basePath)

	info, err := baseFs.Stat(absPath)
	if err != nil {
		return nil, &os.PathError{Op: op, Path: basePath, Err: errors.Unwrap(err)}
	}

	if !info.IsDir() {
		return nil, &os.PathError{Op: op, Path: basePath, Err: avfs.ErrNotADirectory}
	}

	fs := &BasePathFs{
		baseFs:   baseFs,
		basePath: absPath,
	}

	err = fsutil.CreateBaseDirs(fs)
	if err != nil {
		return nil, err
	}

	return fs, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (fs *BasePathFs) Features() avfs.Feature {
	return fs.baseFs.Features() &^ avfs.FeatSymlink
}

// HasFeatures returns true if the file system provides all the given features.
func (fs *BasePathFs) HasFeatures(feature avfs.Feature) bool {
	return (feature&avfs.FeatSymlink == 0) && fs.baseFs.HasFeatures(feature)
}

// Name returns the name of the fileSystem.
func (fs *BasePathFs) Name() string {
	return fs.baseFs.Name()
}

// Type returns the type of the fileSystem or Identity manager.
func (fs *BasePathFs) Type() string {
	return "BasePathFs"
}
