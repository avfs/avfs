//
//  Copyright 2021 The AVFS authors
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

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	// Random number state.
	// We generate random temporary file names so that there's a good
	// chance the file doesn't exist yet - keeps the number of tries in
	// CreateTemp to a minimum.
	randno uint32     //nolint:gochecknoglobals // Used by nextRandom().
	randmu sync.Mutex //nolint:gochecknoglobals // Used by nextRandom().
)

// cleanGlobPath prepares path for glob matching.
func (ut *Utils) cleanGlobPath(path string) string {
	switch path {
	case "":
		return "."
	case string(ut.pathSeparator):
		// do nothing to the path
		return path
	default:
		return path[0 : len(path)-1] // chop off trailing separator
	}
}

// cleanGlobPathWindows is windows version of cleanGlobPath.
func (ut *Utils) cleanGlobPathWindows(path string) (prefixLen int, cleaned string) {
	vollen := ut.volumeNameLen(path)

	switch {
	case path == "":
		return 0, "."
	case vollen+1 == len(path) && ut.IsPathSeparator(path[len(path)-1]): // /, \, C:\ and C:/
		// do nothing to the path
		return vollen + 1, path
	case vollen == len(path) && len(path) == 2: // C:
		return vollen, path + "." // convert C: into C:.
	default:
		if vollen >= len(path) {
			vollen = len(path) - 1
		}

		return vollen, path[0 : len(path)-1] // chop off trailing separator
	}
}

// getEsc gets a possibly-escaped character from chunk, for a character class.
func (ut *Utils) getEsc(chunk string) (r rune, nchunk string, err error) {
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

// glob searches for files matching pattern in the directory dir
// and appends them to matches. If the directory cannot be
// opened, it returns the existing matches. New matches are
// added in lexicographical order.
func (ut *Utils) glob(vfs VFS, dir, pattern string, matches []string) (m []string, e error) {
	m = matches

	fi, err := vfs.Stat(dir)
	if err != nil {
		return // ignore I/O error
	}

	if !fi.IsDir() {
		return // ignore I/O error
	}

	d, err := vfs.Open(dir)
	if err != nil {
		return // ignore I/O error
	}

	defer d.Close()

	names, _ := d.Readdirnames(-1)
	sort.Strings(names)

	for _, n := range names {
		matched, err := ut.Match(pattern, n)
		if err != nil {
			return m, err
		}

		if matched {
			m = append(m, vfs.Join(dir, n))
		}
	}

	return //nolint:nakedret // Adapted from standard library.
}

// hasMeta reports whether path contains any of the magic characters
// recognized by Match.
func (ut *Utils) hasMeta(path string) bool {
	magicChars := `*?[`

	if ut.osType != OsWindows {
		magicChars = `*?[\`
	}

	return strings.ContainsAny(path, magicChars)
}

func (ut *Utils) joinWindows(elem []string) string {
	for i, e := range elem {
		if e != "" {
			return ut.joinNonEmpty(elem[i:])
		}
	}

	return ""
}

// joinNonEmpty is like join, but it assumes that the first element is non-empty.
func (ut *Utils) joinNonEmpty(elem []string) string {
	if len(elem[0]) == 2 && elem[0][1] == ':' {
		// First element is drive letter without terminating slash.
		// Keep path relative to current directory on that drive.
		// Skip empty elements.
		i := 1
		for ; i < len(elem); i++ {
			if elem[i] != "" {
				break
			}
		}

		return ut.Clean(elem[0] + strings.Join(elem[i:], string(ut.pathSeparator)))
	}
	// The following logic prevents Join from inadvertently creating a
	// UNC path on Windows. Unless the first element is a UNC path, Join
	// shouldn't create a UNC path. See golang.org/issue/9167.
	p := ut.Clean(strings.Join(elem, string(ut.pathSeparator)))
	if !ut.isUNC(p) {
		return p
	}

	// p == UNC only allowed when the first element is a UNC path.
	head := ut.Clean(elem[0])
	if ut.isUNC(head) {
		return p
	}

	// head + tail == UNC, but joining two non-UNC paths should not result
	// in a UNC path. Undo creation of UNC path.
	tail := ut.Clean(strings.Join(elem[1:], string(ut.pathSeparator)))
	if head[len(head)-1] == ut.pathSeparator {
		return head + tail
	}

	return head + string(ut.pathSeparator) + tail
}

// isUNC reports whether path is a UNC path.
func (ut *Utils) isUNC(path string) bool {
	return ut.volumeNameLen(path) > 2
}

func (ut *Utils) joinPath(dir, name string) string {
	if len(dir) > 0 && ut.IsPathSeparator(dir[len(dir)-1]) {
		return dir + name
	}

	return dir + string(ut.pathSeparator) + name
}

// reservedNames lists reserved Windows names. Search for PRN in
// https://docs.microsoft.com/en-us/windows/desktop/fileio/naming-a-file
// for details.
var reservedNames = []string{ //nolint:gochecknoglobals // Used by isReservedName().
	"CON", "PRN", "AUX", "NUL",
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
}

// isReservedName returns true, if path is Windows reserved name.
// See reservedNames for the full list.
func isReservedName(path string) bool {
	if path == "" {
		return false
	}

	for _, reserved := range reservedNames {
		if strings.EqualFold(path, reserved) {
			return true
		}
	}

	return false
}

// matchChunk checks whether chunk matches the beginning of s.
// If so, it returns the remainder of s (after the match).
// Chunk is all single-character operators: literals, char classes, and ?.
func (ut *Utils) matchChunk(chunk, s string) (rest string, ok bool, err error) {
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

// scanChunk gets the next segment of pattern, which is a non-star string
// possibly preceded by a star.
func (ut *Utils) scanChunk(pattern string) (star bool, chunk, rest string) {
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

// bufSize is the size of each buffer used to copy files.
const bufSize = 32 * 1024

// bufPool is the buffer pool used to copy files.
var bufPool = &sync.Pool{New: func() interface{} { //nolint:gochecknoglobals // BufPool must be global.
	buf := make([]byte, bufSize)

	return &buf
}}

// copyBufPool copies a source reader to a writer using a buffer from the buffer pool.
func copyBufPool(dst io.Writer, src io.Reader) (written int64, err error) { //nolint:unparam // written is never used.
	buf := bufPool.Get().(*[]byte) //nolint:errcheck // Get() always returns a pointer to a byte slice.
	defer bufPool.Put(buf)

	written, err = io.CopyBuffer(dst, src, *buf)

	return
}

func (ut *Utils) nextRandom() string {
	randmu.Lock()

	r := randno
	if r == 0 {
		r = ut.reseed()
	}

	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	randno = r
	randmu.Unlock()

	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

// prefixAndSuffix splits pattern by the last wildcard "*", if applicable,
// returning prefix as the part before "*" and suffix as the part after "*".
func (ut *Utils) prefixAndSuffix(pattern string) (prefix, suffix string, err error) {
	for i := 0; i < len(pattern); i++ {
		if ut.IsPathSeparator(pattern[i]) {
			return "", "", ErrPatternHasSeparator
		}
	}

	if pos := strings.LastIndexByte(pattern, '*'); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}

	return prefix, suffix, nil
}

func (ut *Utils) reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

// volumeNameLen returns length of the leading volume name on Windows.
// It returns 0 elsewhere.
func (ut *Utils) volumeNameLen(path string) int {
	if ut.osType != OsWindows {
		return 0
	}

	if len(path) < 2 {
		return 0
	}

	// with drive letter
	c := path[0]
	if path[1] == ':' && ('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') { //nolint:gocritic // Adapted from standard library.
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

// walkDir recursively descends path, calling walkDirFn.
func (ut *Utils) walkDir(vfs VFS, path string, d fs.DirEntry, walkDirFn fs.WalkDirFunc) error {
	if err := walkDirFn(path, d, nil); err != nil || !d.IsDir() {
		if err == filepath.SkipDir && d.IsDir() {
			// Successfully skipped directory.
			err = nil
		}

		return err
	}

	dirs, err := vfs.ReadDir(path)
	if err != nil {
		// Second call, to report ReadDir error.
		err = walkDirFn(path, d, err)
		if err != nil {
			return err
		}
	}

	for _, d1 := range dirs {
		path1 := vfs.Join(path, d1.Name())
		if err := ut.walkDir(vfs, path1, d1, walkDirFn); err != nil {
			if err == filepath.SkipDir {
				break
			}

			return err
		}
	}

	return nil
}

func isSlash(c uint8) bool {
	return c == '\\' || c == '/'
}

func sameWord(a, b string) bool {
	return a == b
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
