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

package test

import (
	"io"
	"math/rand"
	"os"
	"strconv"
	"syscall"
	"testing"

	"github.com/avfs/avfs"
)

const (
	bufSize     = 32 * 1024
	maxFileSize = 1024 * bufSize
)

// BenchCreate benchmarks Create function.
func (sfs *SuiteFS) BenchCreate(b *testing.B, testDir string) {
	vfs := sfs.vfsTest

	b.Run("Create", func(b *testing.B) {
		b.StopTimer()

		files := make([]string, b.N)

		for n := 0; n < b.N; n++ {
			path := vfs.Join(testDir, strconv.FormatUint(uint64(n), 10))
			files[n] = path
		}

		b.StartTimer()

		for n := 0; n < b.N; n++ {
			fileName := files[n]

			f, err := vfs.Create(fileName)
			if err != nil {
				b.Fatalf("Create %s : %v", fileName, err)
			}

			_ = f.Close()
		}
	})
}

func (sfs *SuiteFS) BenchFileRead(b *testing.B, testDir string) {
	vfs := sfs.vfsTest
	buf := make([]byte, bufSize)
	fileName := vfs.Join(testDir, "BenchFileRead.txt")

	err := vfs.WriteFile(fileName, make([]byte, maxFileSize), avfs.DefaultFilePerm)
	CheckNoError(b, "WriteFile"+fileName, err)

	f, err := vfs.OpenFile(fileName, os.O_RDONLY|syscall.O_DIRECT, avfs.DefaultFilePerm)
	CheckNoError(b, "OpenFile"+fileName, err)

	defer func() {
		_ = f.Close()
		_ = vfs.Remove(fileName)
	}()

	b.ResetTimer()

	b.Run("FileRead", func(b *testing.B) {
		_, err = f.Seek(0, io.SeekStart)
		CheckNoError(b, "Seek"+fileName, err)

		s := 0

		for n := 0; n < b.N; n++ {
			if s >= maxFileSize {
				s = 0
				_, err = f.Seek(0, io.SeekStart)
				CheckNoError(b, "Seek"+fileName, err)
			}

			_, err = f.Read(buf)
			CheckNoError(b, "Read"+fileName, err)

			s += bufSize
		}
	})
}

func (sfs *SuiteFS) BenchFileWrite(b *testing.B, testDir string) {
	vfs := sfs.vfsTest
	buf := make([]byte, bufSize)
	fileName := vfs.Join(testDir, "BenchFileWrite.txt")

	f, err := vfs.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|syscall.O_DIRECT, avfs.DefaultFilePerm)
	CheckNoError(b, "OpenFile"+fileName, err)

	defer func() {
		_ = f.Close()
		_ = vfs.Remove(fileName)
	}()

	b.ResetTimer()

	b.Run("FileWrite", func(b *testing.B) {
		s := 0

		for n := 0; n < b.N; n++ {
			s += bufSize
			if s >= maxFileSize {
				s = 0
				_, err = f.Seek(0, io.SeekStart)
				if err != nil {
					b.Fatalf("Seek %s : %v", fileName, err)
				}
			}

			_, err = f.Write(buf)
			if err != nil {
				b.Fatalf("Write %s : %v", fileName, err)
			}
		}
	})
}

// BenchMkdir benchmarks Mkdir function.
func (sfs *SuiteFS) BenchMkdir(b *testing.B, testDir string) {
	vfs := sfs.vfsTest

	b.Run("Mkdir", func(b *testing.B) {
		b.StopTimer()
		dirs := make([]string, b.N)
		dirs[0] = testDir

		for n := 1; n < b.N; n++ {
			parent := dirs[rand.Intn(n)]
			path := vfs.Join(parent, strconv.FormatUint(rand.Uint64(), 10))
			dirs[n] = path
		}

		err := vfs.RemoveAll(testDir)
		if err != nil {
			b.Fatalf("RemoveAll %s : %v", testDir, err)
		}

		b.StartTimer()

		for n := 0; n < b.N; n++ {
			err = vfs.Mkdir(dirs[n], avfs.DefaultDirPerm)
			if err != nil {
				b.Fatalf("Mkdir %s : %v", dirs[n], err)
			}
		}
	})
}

// BenchOpenFile benchmarks OpenFile function.
func (sfs *SuiteFS) BenchOpenFile(b *testing.B, testDir string) {
	vfs := sfs.vfsTest
	fileName := sfs.existingFile(b, testDir, nil)

	b.ResetTimer()

	b.Run("Open", func(b *testing.B) {
		for n := 1; n < b.N; n++ {
			f, err := vfs.OpenFile(fileName, os.O_RDONLY, avfs.DefaultFilePerm)
			if err != nil {
				b.Fatalf("OpenFile %s : %v", fileName, err)
			}

			_ = f.Close()
		}
	})
}

// BenchRemove benchmarks Remove function.
func (sfs *SuiteFS) BenchRemove(b *testing.B, testDir string) {
	vfs := sfs.vfsTest

	b.Run("Remove", func(b *testing.B) {
		b.StopTimer()

		rt, err := avfs.NewRndTree(vfs, testDir, &avfs.RndTreeParams{
			MinName: 10,
			MaxName: 10,
			MinDirs: b.N,
			MaxDirs: b.N,
		})

		if !CheckNoError(b, "RndTree "+testDir, err) {
			return
		}

		err = rt.CreateTree()
		if !CheckNoError(b, "CreateTree "+testDir, err) {
			return
		}

		b.StartTimer()

		for n := b.N - 1; n >= 0; n-- {
			err = vfs.Remove(rt.Dirs[n])
			if err != nil {
				b.Fatalf("Remove %s : %v", rt.Dirs[n], err)
			}
		}
	})
}
