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

package avfs

// CurDirMgr is the interface that manages the current directory.
type CurDirMgr interface {
	// CurDir returns the current directory.
	CurDir() string

	// SetCurDir sets the current directory.
	SetCurDir(curDir string) error
}

// CurDirFn provides current directory functions to a file system.
type CurDirFn struct {
	curDir string // curDir is the current directory.
}

// CurDir returns the current directory.
func (cdf *CurDirFn) CurDir() string {
	return cdf.curDir
}

// SetCurDir sets the current directory.
func (cdf *CurDirFn) SetCurDir(curDir string) error {
	cdf.curDir = curDir

	return nil
}
