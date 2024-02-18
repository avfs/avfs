//
//  Copyright 2023 The AVFS authors
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

//go:build !avfs_setostype

package avfs

import (
	"os"
	"path/filepath"
)

const buildFeatSetOSType = 0

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (*VFSFn[_]) Base(path string) string {
	return filepath.Base(path)
}

// Clean returns the shortest path name equivalent to path
// by purely lexical processing. It applies the following rules
// iteratively until no further processing can be done:
//
//  1. Replace multiple Separator elements with a single one.
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
// See also Rob Pike, “Lexical File Names in Plan 9 or
// Getting Dot-Dot Right,”
// https://9p.io/sys/doc/lexnames.html
func (*VFSFn[_]) Clean(path string) string {
	return filepath.Clean(path)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (*VFSFn[_]) Dir(path string) string {
	return filepath.Dir(path)
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (*VFSFn[_]) FromSlash(path string) string {
	return filepath.FromSlash(path)
}

// IsAbs reports whether the path is absolute.
func (*VFSFn[_]) IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

// IsPathSeparator reports whether c is a directory separator character.
func (*VFSFn[_]) IsPathSeparator(c uint8) bool {
	return os.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path,
// separating them with an OS specific Separator. Empty elements
// are ignored. The result is Cleaned. However, if the argument
// list is empty or all its elements are empty, Join returns
// an empty string.
// On Windows, the result will only be a UNC path if the first
// non-empty element is a UNC path.
func (*VFSFn[_]) Join(elem ...string) string {
	return filepath.Join(elem...)
}

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
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
//
// On Windows, escaping is disabled. Instead, '\\' is treated as
// path separator.
func (*VFSFn[_]) Match(pattern, name string) (matched bool, err error) {
	return filepath.Match(pattern, name)
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (*VFSFn[_]) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (*VFSFn[_]) Split(path string) (dir, file string) {
	return filepath.Split(path)
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (*VFSFn[_]) ToSlash(path string) string {
	return filepath.ToSlash(path)
}
