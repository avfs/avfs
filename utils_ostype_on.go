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

//go:build avfs_setostype

package avfs

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const buildFeatSetOSType = FeatSetOSType

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (ut *Utils[_]) Base(path string) string {
	if path == "" {
		return "."
	}

	// Strip trailing slashes.
	for len(path) > 0 && ut.IsPathSeparator(path[len(path)-1]) {
		path = path[0 : len(path)-1]
	}

	// Throw away volume name
	path = path[len(ut.VolumeName(path)):]

	// Find the last element
	i := len(path) - 1
	for i >= 0 && !ut.IsPathSeparator(path[i]) {
		i--
	}

	if i >= 0 {
		path = path[i+1:]
	}

	// If empty now, it had only slashes.
	if path == "" {
		return string(ut.pathSeparator)
	}

	return path
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
// On Windows, Clean does not modify the volume name other than to replace
// occurrences of "/" with `\`.
// For example, Clean("//host/share/../x") returns `\\host\share\x`.
//
// See also Rob Pike, “Lexical File Names in Plan 9 or
// Getting Dot-Dot Right,”
// https://9p.io/sys/doc/lexnames.html
func (ut *Utils[_]) Clean(path string) string {
	if ut.osType == CurrentOSType() {
		return filepath.Clean(path)
	}

	originalPath := path
	volLen := ut.VolumeNameLen(path)

	path = path[volLen:]
	if path == "" {
		if volLen > 1 && ut.IsPathSeparator(originalPath[0]) && ut.IsPathSeparator(originalPath[1]) {
			// should be UNC
			return ut.FromSlash(originalPath)
		}

		return originalPath + "."
	}

	rooted := ut.IsPathSeparator(path[0])

	// Invariants:
	//	reading from path; r is index of next byte to process.
	//	writing to buf; w is index of next byte to write.
	//	dotdot is index in buf where .. must stop, either because
	//		it is the leading slash or it is a leading ../../.. prefix.
	n := len(path)
	out := lazybuf{path: path, volAndPath: originalPath, volLen: volLen}
	r, dotdot := 0, 0

	if rooted {
		out.append(ut.pathSeparator)

		r, dotdot = 1, 1
	}

	for r < n {
		switch {
		case ut.IsPathSeparator(path[r]):
			// empty path element
			r++
		case path[r] == '.' && (r+1 == n || ut.IsPathSeparator(path[r+1])):
			// . element
			r++
		case path[r] == '.' && path[r+1] == '.' && (r+2 == n || ut.IsPathSeparator(path[r+2])):
			// .. element: remove to last separator
			r += 2

			switch {
			case out.w > dotdot:
				// can backtrack
				out.w--
				for out.w > dotdot && !ut.IsPathSeparator(out.index(out.w)) {
					out.w--
				}
			case !rooted:
				// cannot backtrack, but not rooted, so append .. element.
				if out.w > 0 {
					out.append(ut.pathSeparator)
				}

				out.append('.')
				out.append('.')
				dotdot = out.w
			}
		default:
			// real path element.
			// add slash if needed
			if rooted && out.w != 1 || !rooted && out.w != 0 {
				out.append(ut.pathSeparator)
			}

			// If a ':' appears in the path element at the start of a Windows path,
			// insert a .\ at the beginning to avoid converting relative paths
			// like a/../c: into c:.
			if ut.osType == OsWindows && out.w == 0 && out.volLen == 0 && r != 0 {
				for i := r; i < n && !ut.IsPathSeparator(path[i]); i++ {
					if path[i] == ':' {
						out.append('.')
						out.append(ut.pathSeparator)

						break
					}
				}
			}

			// copy element
			for ; r < n && !ut.IsPathSeparator(path[r]); r++ {
				out.append(path[r])
			}
		}
	}

	// Turn empty string into "."
	if out.w == 0 {
		out.append('.')
	}

	return ut.FromSlash(out.string())
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (ut *Utils[_]) Dir(path string) string {
	vol := ut.VolumeName(path)

	i := len(path) - 1
	for i >= len(vol) && !ut.IsPathSeparator(path[i]) {
		i--
	}

	dir := ut.Clean(path[len(vol) : i+1])
	if dir == "." && len(vol) > 2 {
		// must be UNC
		return vol
	}

	return vol + dir
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (ut *Utils[_]) FromSlash(path string) string {
	if ut.osType != OsWindows {
		return path
	}

	return strings.ReplaceAll(path, "/", string(ut.pathSeparator))
}

// getEsc gets a possibly-escaped character from chunk, for a character class.
func (ut *Utils[_]) getEsc(chunk string) (r rune, nchunk string, err error) {
	if chunk == "" || chunk[0] == '-' || chunk[0] == ']' {
		err = filepath.ErrBadPattern

		return
	}

	if chunk[0] == '\\' && ut.osType != OsWindows {
		chunk = chunk[1:]
		if chunk == "" {
			err = filepath.ErrBadPattern

			return
		}
	}

	r, n := utf8.DecodeRuneInString(chunk)
	if r == utf8.RuneError && n == 1 {
		err = filepath.ErrBadPattern
	}

	nchunk = chunk[n:]
	if nchunk == "" {
		err = filepath.ErrBadPattern
	}

	return
}

// IsAbs reports whether the path is absolute.
func (ut *Utils[_]) IsAbs(path string) bool {
	if ut.osType != OsWindows {
		return strings.HasPrefix(path, "/")
	}

	l := ut.VolumeNameLen(path)
	if l == 0 {
		return false
	}

	// If the volume name starts with a double slash, this is an absolute path.
	if isSlash(path[0]) && isSlash(path[1]) {
		return true
	}

	path = path[l:]
	if path == "" {
		return false
	}

	return isSlash(path[0])
}

// IsPathSeparator reports whether c is a directory separator character.
func (ut *Utils[_]) IsPathSeparator(c uint8) bool {
	if ut.osType != OsWindows {
		return c == '/'
	}

	return c == '\\' || c == '/'
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (ut *Utils[_]) Join(elem ...string) string {
	if ut.osType != OsWindows {
		// If there's a bug here, fix the logic in ./path_plan9.go too.
		for i, e := range elem {
			if e != "" {
				return ut.Clean(strings.Join(elem[i:], string(ut.pathSeparator)))
			}
		}

		return ""
	}

	return ut.joinWindows(elem)
}

func isSlash(c uint8) bool {
	return c == '\\' || c == '/'
}

func (ut *Utils[_]) joinWindows(elem []string) string {
	var (
		b        strings.Builder
		lastChar byte
	)

	for _, e := range elem {
		switch {
		case b.Len() == 0:
			// Add the first non-empty path element unchanged.
		case isSlash(lastChar):
			// If the path ends in a slash, strip any leading slashes from the next
			// path element to avoid creating a UNC path (any path starting with "\\")
			// from non-UNC elements.
			//
			// The correct behavior for Join when the first element is an incomplete UNC
			// path (for example, "\\") is underspecified. We currently join subsequent
			// elements so Join("\\", "host", "share") produces "\\host\share".
			for len(e) > 0 && isSlash(e[0]) {
				e = e[1:]
			}
		case lastChar == ':':
			// If the path ends in a colon, keep the path relative to the current directory
			// on a drive and don't add a separator. Preserve leading slashes in the next
			// path element, which may make the path absolute.
			//
			// 	Join(`C:`, `f`) = `C:f`
			//	Join(`C:`, `\f`) = `C:\f`
		default:
			// In all other cases, add a separator between elements.
			b.WriteByte('\\')

			lastChar = '\\'
		}

		if len(e) > 0 {
			b.WriteString(e)
			lastChar = e[len(e)-1]
		}
	}

	if b.Len() == 0 {
		return ""
	}

	return ut.Clean(b.String())
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
func (ut *Utils[_]) Match(pattern, name string) (matched bool, err error) {
Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string

		star, chunk, pattern = ut.scanChunk(pattern)
		if star && chunk == "" {
			// Trailing * matches rest of string unless it has a /.
			return !strings.Contains(name, string(ut.pathSeparator)), nil
		}

		// Look for match at current position.
		t, ok, err := ut.matchChunk(chunk, name)

		// if we're the last chunk, make sure we've exhausted the name
		// otherwise we'll give a false result even if we could still match
		// using the star
		if ok && (t == "" || len(pattern) > 0) {
			name = t

			continue
		}

		if err != nil {
			return false, err
		}

		if star {
			// Look for match skipping i+1 bytes.
			// Cannot skip /.
			for i := 0; i < len(name) && name[i] != ut.pathSeparator; i++ {
				t, ok, err := ut.matchChunk(chunk, name[i+1:])
				if ok {
					// if we're the last chunk, make sure we exhausted the name
					if pattern == "" && len(t) > 0 {
						continue
					}
					name = t

					continue Pattern
				}
				if err != nil {
					return false, err
				}
			}
		}

		return false, nil
	}

	return name == "", nil
}

// matchChunk checks whether chunk matches the beginning of s.
// If so, it returns the remainder of s (after the match).
// Chunk is all single-character operators: literals, char classes, and ?.
func (ut *Utils[_]) matchChunk(chunk, s string) (rest string, ok bool, err error) {
	// failed records whether the match has failed.
	// After the match fails, the loop continues on processing chunk,
	// checking that the pattern is well-formed but no longer reading s.
	failed := false

	for len(chunk) > 0 {
		if !failed && s == "" {
			failed = true
		}

		switch chunk[0] {
		case '[':
			// character class
			var r rune

			if !failed {
				var n int
				r, n = utf8.DecodeRuneInString(s)
				s = s[n:]
			}

			chunk = chunk[1:]
			// possibly negated
			negated := false

			if len(chunk) > 0 && chunk[0] == '^' {
				negated = true
				chunk = chunk[1:]
			}

			// parse all ranges
			match := false
			nrange := 0

			for {
				if len(chunk) > 0 && chunk[0] == ']' && nrange > 0 {
					chunk = chunk[1:]

					break
				}

				var lo, hi rune

				if lo, chunk, err = ut.getEsc(chunk); err != nil {
					return "", false, err
				}

				hi = lo

				if chunk[0] == '-' {
					if hi, chunk, err = ut.getEsc(chunk[1:]); err != nil {
						return "", false, err
					}
				}

				if lo <= r && r <= hi {
					match = true
				}

				nrange++
			}

			if match == negated {
				failed = true
			}
		case '?':
			if !failed {
				if s[0] == ut.pathSeparator {
					failed = true
				}

				_, n := utf8.DecodeRuneInString(s)
				s = s[n:]
			}

			chunk = chunk[1:]
		case '\\':
			if ut.osType != OsWindows {
				chunk = chunk[1:]
				if chunk == "" {
					return "", false, filepath.ErrBadPattern
				}
			}

			fallthrough
		default:
			if !failed {
				if chunk[0] != s[0] {
					failed = true
				}

				s = s[1:]
			}

			chunk = chunk[1:]
		}
	}

	if failed {
		return "", false, nil
	}

	return s, true, nil
}

// PathSeparator return the OS-specific path separator.
func (ut *Utils[_]) PathSeparator() uint8 {
	return ut.pathSeparator
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (ut *Utils[_]) Rel(basepath, targpath string) (string, error) {
	baseVol := ut.VolumeName(basepath)
	targVol := ut.VolumeName(targpath)
	base := ut.Clean(basepath)
	targ := ut.Clean(targpath)

	if ut.sameWord(targ, base) {
		return ".", nil
	}

	base = base[len(baseVol):]
	targ = targ[len(targVol):]

	if base == "." {
		base = ""
	} else if base == "" && ut.VolumeNameLen(baseVol) > 2 /* isUNC */ {
		// Treat any targetpath matching `\\host\share` basepath as absolute path.
		base = string(ut.pathSeparator)
	}

	// Can't use IsAbs - `\a` and `a` are both relative in Windows.
	baseSlashed := len(base) > 0 && base[0] == ut.pathSeparator
	targSlashed := len(targ) > 0 && targ[0] == ut.pathSeparator

	if baseSlashed != targSlashed || !ut.sameWord(baseVol, targVol) {
		return "", errors.New("Rel: can't make " + targpath + " relative to " + basepath)
	}

	// Position base[b0:bi] and targ[t0:ti] at the first differing elements.
	bl := len(base)
	tl := len(targ)

	var b0, bi, t0, ti int

	for {
		for bi < bl && base[bi] != ut.pathSeparator {
			bi++
		}

		for ti < tl && targ[ti] != ut.pathSeparator {
			ti++
		}

		if !ut.sameWord(targ[t0:ti], base[b0:bi]) {
			break
		}

		if bi < bl {
			bi++
		}

		if ti < tl {
			ti++
		}

		b0 = bi
		t0 = ti
	}

	if base[b0:bi] == ".." {
		return "", errors.New("Rel: can't make " + targpath + " relative to " + basepath)
	}

	if b0 != bl {
		// Base elements left. Must go up before going down.
		seps := strings.Count(base[b0:bl], string(ut.pathSeparator))
		size := 2 + seps*3

		if tl != t0 {
			size += 1 + tl - t0
		}

		buf := make([]byte, size)
		n := copy(buf, "..")

		for i := 0; i < seps; i++ {
			buf[n] = ut.pathSeparator
			copy(buf[n+1:], "..")
			n += 3
		}

		if t0 != tl {
			buf[n] = ut.pathSeparator
			copy(buf[n+1:], targ[t0:])
		}

		return string(buf), nil
	}

	return targ[t0:], nil
}

func (ut *Utils[_]) sameWord(a, b string) bool {
	if ut.osType != OsWindows {
		return a == b
	}

	return strings.EqualFold(a, b)
}

// scanChunk gets the next segment of pattern, which is a non-star string
// possibly preceded by a star.
func (ut *Utils[_]) scanChunk(pattern string) (star bool, chunk, rest string) {
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true
	}

	inrange := false

	var i int

Scan:
	for i = 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			if ut.osType != OsWindows {
				// error check handled in matchChunk: bad pattern.
				if i+1 < len(pattern) {
					i++
				}
			}
		case '[':
			inrange = true
		case ']':
			inrange = false
		case '*':
			if !inrange {
				break Scan
			}
		}
	}

	return star, pattern[0:i], pattern[i:]
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (ut *Utils[_]) Split(path string) (dir, file string) {
	vol := ut.VolumeName(path)

	i := len(path) - 1
	for i >= len(vol) && !ut.IsPathSeparator(path[i]) {
		i--
	}

	return path[:i+1], path[i+1:]
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (ut *Utils[_]) ToSlash(path string) string {
	if ut.pathSeparator == '/' {
		return path
	}

	return strings.ReplaceAll(path, string(ut.pathSeparator), "/")
}

// VolumeName returns leading volume name.
// Given "C:\foo\bar" it returns "C:" on Windows.
// Given "\\host\share\foo" it returns "\\host\share".
// On other platforms it returns "".
func (ut *Utils[_]) VolumeName(path string) string {
	return ut.FromSlash(path[:ut.VolumeNameLen(path)])
}

// VolumeNameLen returns length of the leading volume name on Windows.
// It returns 0 elsewhere.
func (ut *Utils[_]) VolumeNameLen(path string) int {
	if ut.osType != OsWindows {
		return 0
	}

	if len(path) < 2 {
		return 0
	}

	// with drive letter
	c := path[0]
	if path[1] == ':' && ('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
		return 2
	}

	// is it UNC? https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	if l := len(path); l >= 5 && isSlash(path[0]) && isSlash(path[1]) &&
		!isSlash(path[2]) && path[2] != '.' {
		// first, leading `\\` and next shouldn't be `\`. its server name.
		for n := 3; n < l-1; n++ {
			// second, next '\' shouldn't be repeated.
			if isSlash(path[n]) {
				n++
				// third, following something characters. its share name.
				if !isSlash(path[n]) {
					if path[n] == '.' {
						break
					}

					for ; n < l; n++ {
						if isSlash(path[n]) {
							break
						}
					}

					return n
				}

				break
			}
		}
	}

	return 0
}

// A lazybuf is a lazily constructed path buffer.
// It supports append, reading previously appended bytes,
// and retrieving the final string. It does not allocate a buffer
// to hold the output until that output diverges from s.
type lazybuf struct {
	path       string
	volAndPath string
	buf        []byte
	w          int
	volLen     int
}

func (b *lazybuf) index(i int) byte {
	if b.buf != nil {
		return b.buf[i]
	}

	return b.path[i]
}

func (b *lazybuf) append(c byte) {
	if b.buf == nil {
		if b.w < len(b.path) && b.path[b.w] == c {
			b.w++

			return
		}

		b.buf = make([]byte, len(b.path))
		copy(b.buf, b.path[:b.w])
	}

	b.buf[b.w] = c
	b.w++
}

func (b *lazybuf) string() string {
	if b.buf == nil {
		return b.volAndPath[:b.volLen+b.w]
	}

	return b.volAndPath[:b.volLen] + string(b.buf[:b.w])
}
