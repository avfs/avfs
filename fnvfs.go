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

// FnVFS defines the function names of a virtual file system that can return an error (see failfs.FailFS).
type FnVFS uint

//go:generate stringer -type FnVFS -trimprefix Fn -output fnvfs_string.go

const (
	FnAbs FnVFS = iota + 1
	FnChdir
	FnChmod
	FnChown
	FnChtimes
	FnCreateTemp
	FnEvalSymlinks
	FnFileChdir
	FnFileChmod
	FnFileChown
	FnFileClose
	FnFileRead
	FnFileReadAt
	FnFileReadDir
	FnFileReaddirnames
	FnFileSeek
	FnFileStat
	FnFileSync
	FnFileTruncate
	FnFileWrite
	FnFileWriteAt
	FnGetwd
	FnLchown
	FnLink
	FnLstat
	FnMkdir
	FnMkdirAll
	FnMkdirTemp
	FnOpenFile
	FnReadDir
	FnReadFile
	FnReadlink
	FnRemove
	FnRemoveAll
	FnRename
	FnSetUser
	FnSetUserByName
	FnStat
	FnSub
	FnSymlink
	FnTruncate
	FnWalkDir
	FnWriteFile
)
