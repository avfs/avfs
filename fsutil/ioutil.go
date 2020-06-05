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

package fsutil

import (
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/avfs/avfs"
)

var (
	// Random number state.
	// We generate random temporary file names so that there's a good
	// chance the file doesn't exist yet - keeps the number of tries in
	// TempFile to a minimum.
	randno uint32     //nolint:gochecknoglobals
	randmu sync.Mutex //nolint:gochecknoglobals
)

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextRandom() string {
	randmu.Lock()

	r := randno
	if r == 0 {
		r = reseed()
	}

	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	randno = r
	randmu.Unlock()

	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

// ReadDir reads the directory named by dirname and returns
// a list of directory entries sorted by filename.
func ReadDir(fs avfs.Fs, dirname string) ([]os.FileInfo, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}

	list, err := f.Readdir(-1)
	_ = f.Close()

	if err != nil {
		return nil, err
	}

	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })

	return list, nil
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func ReadFile(fs avfs.Fs, filename string) ([]byte, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return ioutil.ReadAll(f)
}

// prefixAndSuffix splits pattern by the last wildcard "*", if applicable,
// returning prefix as the part before "*" and suffix as the part after "*".
func prefixAndSuffix(pattern string) (prefix, suffix string) {
	if pos := strings.LastIndex(pattern, "*"); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}

	return
}

// TempDir creates a new temporary directory in the directory dir.
// The directory name is generated by taking pattern and applying a
// random string to the end. If pattern includes a "*", the random string
// replaces the last "*". TempDir returns the name of the new directory.
// If dir is the empty string, TempDir uses the
// default directory for temporary files (see os.TempDir).
// Multiple programs calling TempDir simultaneously
// will not choose the same directory. It is the caller's responsibility
// to remove the directory when no longer needed.
func TempDir(fs avfs.Fs, dir, pattern string) (name string, err error) {
	if dir == "" {
		dir = fs.GetTempDir()
	}

	prefix, suffix := prefixAndSuffix(pattern)
	nconflict := 0

	for i := 0; i < 10000; i++ {
		try := Join(dir, prefix+nextRandom()+suffix)
		err = fs.Mkdir(try, 0o700)

		if fs.IsExist(err) {
			nconflict++
			if nconflict > 10 {
				randmu.Lock()
				randno = reseed()
				randmu.Unlock()
			}

			continue
		}

		if fs.IsNotExist(err) {
			if _, err1 := fs.Stat(dir); fs.IsNotExist(err) {
				return "", err1
			}
		}

		if err == nil {
			name = try
		}

		break
	}

	return //nolint:nakedret
}

// TempFile creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting *os.File.
// The filename is generated by taking pattern and adding a random
// string to the end. If pattern includes a "*", the random string
// replaces the last "*".
// If dir is the empty string, TempFile uses the default directory
// for temporary files (see os.TempDir).
// Multiple programs calling TempFile simultaneously
// will not choose the same file. The caller can use f.Name()
// to find the pathname of the file. It is the caller's responsibility
// to remove the file when no longer needed.
func TempFile(fs avfs.Fs, dir, pattern string) (f avfs.File, err error) {
	if dir == "" {
		dir = fs.GetTempDir()
	}

	prefix, suffix := prefixAndSuffix(pattern)

	nconflict := 0

	for i := 0; i < 10000; i++ {
		name := Join(dir, prefix+nextRandom()+suffix)
		f, err = fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)

		if fs.IsExist(err) {
			nconflict++
			if nconflict > 10 {
				randmu.Lock()
				randno = reseed()
				randmu.Unlock()
			}

			continue
		}

		break
	}

	return
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func WriteFile(fs avfs.Fs, filename string, data []byte, perm os.FileMode) error {
	f, err := fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}

	if err1 := f.Close(); err == nil {
		err = err1
	}

	return err
}
