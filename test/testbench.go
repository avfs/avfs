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
	"math/rand"
	"strconv"
	"testing"

	"github.com/avfs/avfs"
)

func (sfs *SuiteFS) BenchAll(b *testing.B) {
	sfs.RunBenches(b, UsrTest,
		sfs.BenchCreate,
		sfs.BenchMkdir,
		sfs.BenchOpenFile,
		sfs.BenchRemove,
		sfs.BenchStat)
}

// BenchCreate benchmarks Create function.
func (sfs *SuiteFS) BenchCreate(b *testing.B, testDir string) {
	vfs := sfs.vfsTest

	b.Run("Create", func(b *testing.B) {
		files := make([]string, b.N)

		for n := 0; n < b.N; n++ {
			path := vfs.Join(testDir, strconv.FormatUint(uint64(n), 10))
			files[n] = path
		}

		sfs.BenchStart(b)
		defer sfs.BenchStop(b)

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

// BenchMkdir benchmarks Mkdir function.
func (sfs *SuiteFS) BenchMkdir(b *testing.B, testDir string) {
	vfs := sfs.vfsTest

	b.Run("Mkdir", func(b *testing.B) {
		dirs := make([]string, b.N)
		dirs[0] = testDir

		for n := 1; n < b.N; n++ {
			parent := dirs[rand.Intn(n)]
			path := vfs.Join(parent, strconv.FormatUint(rand.Uint64(), 10))
			dirs[n] = path
		}

		sfs.BenchStart(b)
		defer sfs.BenchStop(b)

		for n := 0; n < b.N; n++ {
			err := vfs.Mkdir(dirs[n], avfs.DefaultDirPerm)
			if err != nil {
				b.Fatalf("Mkdir %s : %v", dirs[n], err)
			}
		}
	})
}

// BenchOpenFile benchmarks OpenFile function.
func (sfs *SuiteFS) BenchOpenFile(b *testing.B, testDir string) {
}

// BenchRemove benchmarks Remove function.
func (sfs *SuiteFS) BenchRemove(b *testing.B, testDir string) {
	vfs := sfs.vfsTest

	b.Run("Remove", func(b *testing.B) {
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

		sfs.BenchStart(b)
		defer sfs.BenchStop(b)

		for n := b.N - 1; n >= 0; n-- {
			err = vfs.Remove(rt.Dirs[n])
			if err != nil {
				b.Fatalf("Remove %s : %v", rt.Dirs[n], err)
			}
		}
	})
}

// BenchStat benchmarks OpenFile function.
func (sfs *SuiteFS) BenchStat(b *testing.B, testDir string) {
}
