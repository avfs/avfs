//
//  Copyright 2026 The AVFS authors
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

import "io/fs"

// VFSPath defines an interface for OS-specific path operations.
type VFSPath interface {
	// Base returns the last element of path.
	// Trailing path separators are removed before extracting the last element.
	// If the path is empty, Base returns ".".
	// If the path consists entirely of separators, Base returns a single separator.
	Base(path string) string

	// Clean returns the shortest path name equivalent to path
	// by purely lexical processing. It applies the following rules
	// iteratively until no further processing can be done:
	//
	//  1. Replace multiple [Separator] elements with a single one.
	//  2. Eliminate each . path name element (the current directory).
	//  3. Eliminate each inner .. path name element (the parent directory)
	//     along with the non-.. element that precedes it.
	//  4. Eliminate .. elements that begin a rooted path:
	//     that is, replace "/.." by "/" at the beginning of a path,
	//     assuming Separator is '/'.
	//
	// The returned path ends in a slash only if it represents a root directory,
	// such as "/" on Unix or `C:\` on Windows.
	//
	// Finally, any occurrences of slash are replaced by Separator.
	//
	// If the result of this process is an empty string, Clean
	// returns the string ".".
	//
	// On Windows, Clean does not modify the volume name other than to replace
	// occurrences of "/" with `\`.
	// For example, Clean("//host/share/../x") returns `\\host\share\x`.
	//
	// See also Rob Pike, “Lexical File Names in Plan 9 or
	// Getting Dot-Dot Right,”
	// https://9p.io/sys/doc/lexnames.html
	Clean(path string) string

	// Dir returns all but the last element of path, typically the path's directory.
	// After dropping the final element, Dir calls [Clean] on the path and trailing
	// slashes are removed.
	// If the path is empty, Dir returns ".".
	// If the path consists entirely of separators, Dir returns a single separator.
	// The returned path does not end in a separator unless it is the root directory.
	Dir(path string) string

	// FromSlash returns the result of replacing each slash ('/') character
	// in path with a separator character. Multiple slashes are replaced
	// by multiple separators.
	//
	// See also the Localize function, which converts a slash-separated path
	// as used by the io/fs package to an operating system path.
	FromSlash(path string) string

	// IsAbs reports whether the path is absolute.
	IsAbs(path string) bool

	// IsPathSeparator reports whether c is a directory separator character.
	IsPathSeparator(c uint8) bool

	// Join joins any number of path elements into a single path,
	// separating them with an OS specific Separator. Empty elements
	// are ignored. The result is Cleaned. However, if the argument
	// list is empty or all its elements are empty, Join returns
	// an empty string.
	// On Windows, the result will only be a UNC path if the first
	// non-empty element is a UNC path.
	Join(elem ...string) string

	// Match reports whether name matches the shell file name pattern.
	// The pattern syntax is:
	//
	//	pattern:
	//		{ term }
	//	term:
	//		'*'         matches any sequence of non-Separator characters
	//		'?'         matches any single non-Separator character
	//		'[' [ '^' ] { character-range } ']'
	//		            character class (must be non-empty)
	//		c           matches character c (c != '*', '?', '\\', '[')
	//		'\\' c      matches character c
	//
	//	character-range:
	//		c           matches character c (c != '\\', '-', ']')
	//		'\\' c      matches character c
	//		lo '-' hi   matches character c for lo <= c <= hi
	//
	// Match requires pattern to match all of name, not just a substring.
	// The only possible returned error is [ErrBadPattern], when pattern
	// is malformed.
	//
	// On Windows, escaping is disabled. Instead, '\\' is treated as
	// path separator.
	Match(pattern, name string) (matched bool, err error)

	// PathSeparator return the OS-specific path separator.
	PathSeparator() uint8

	// Rel returns a relative path that is lexically equivalent to targpath when
	// joined to basepath with an intervening separator. That is,
	// [Join](basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
	// On success, the returned path will always be relative to basepath,
	// even if basepath and targpath share no elements.
	// An error is returned if targpath can't be made relative to basepath or if
	// knowing the current working directory would be necessary to compute it.
	// Rel calls [Clean] on the result.
	Rel(basepath, targpath string) (string, error)

	// Split splits path immediately following the final [Separator],
	// separating it into a directory and file name component.
	// If there is no Separator in path, Split returns an empty dir
	// and file set to path.
	// The returned values have the property that path = dir+file.
	Split(path string) (dir, file string)

	// ToSlash returns the result of replacing each separator character
	// in path with a slash ('/') character. Multiple separators are
	// replaced by multiple slashes.
	ToSlash(path string) string

	// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
	ToSysStat(info fs.FileInfo) SysStater

	// VolumeName returns the leading volume name.
	// Given "C:\foo\bar" it returns "C:" on Windows.
	// Given "\\host\share\foo" it returns "\\host\share".
	// On other platforms it returns "".
	VolumeName(path string) string

	// VolumeNameLen returns the length of the leading volume name on Windows.
	// It returns 0 elsewhere.
	VolumeNameLen(path string) int
}

// VFSPathFn provides OS-specific path operations.
type VFSPathFn struct {
	FeaturesFn
	OSTypeFn
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vof *VFSPathFn) ToSysStat(info fs.FileInfo) SysStater {
	return info.Sys().(SysStater) //nolint:forcetypeassert // type assertion must be checked
}

// VolumeManager is the interface that manage volumes for Windows file systems.
type VolumeManager interface {
	// VolumeAdd adds a new volume to a Windows file system.
	// If there is an error, it will be of type *PathError.
	VolumeAdd(name string) error

	// VolumeDelete deletes an existing volume and all its files from a Windows file system.
	// If there is an error, it will be of type *PathError.
	VolumeDelete(name string) error

	// VolumeList returns the volumes of the file system.
	VolumeList() []string
}
