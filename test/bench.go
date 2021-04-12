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
	sfs.RunBenchs(b, UsrTest, sfs.BenchCreate)
	sfs.RunBenchs(b, UsrTest, sfs.BenchMkdir)
}

// BenchMkdir benchmarks Mkdir function.
func (sfs *SuiteFS) BenchMkdir(b *testing.B, testDir string) {
	vfs := sfs.vfsTest

	b.Run("Mkdir", func(b *testing.B) {
		dirs := make([]string, 0, b.N)
		dirs = append(dirs, testDir)

		for n := 0; n < b.N; n++ {
			nbDirs := int32(len(dirs))
			parent := dirs[rand.Int31n(nbDirs)]                             //nolint:gosec // No security-sensitive function.
			path := vfs.Join(parent, strconv.FormatUint(rand.Uint64(), 10)) //nolint:gosec // No security-sensitive function.
			dirs = append(dirs, path)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for n := 0; n < b.N; n++ {
			_ = vfs.Mkdir(dirs[n], avfs.DefaultDirPerm)
		}
	})
}

// BenchCreate benchmarks Create function.
func (sfs *SuiteFS) BenchCreate(b *testing.B, testDir string) {
	vfs := sfs.vfsTest

	b.Run("Create", func(b *testing.B) {
		files := make([]string, 0, b.N)

		for n := 0; n < b.N; n++ {
			path := vfs.Join(testDir, strconv.FormatInt(int64(n), 10))
			files = append(files, path)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for n := 0; n < b.N; n++ {
			fileName := files[n]
			f, err := vfs.Create(fileName)
			if err != nil {
				b.Fatalf("Create %s, want error to be nil, got %v", fileName, err)
			}

			err = f.Close()
			if err != nil {
				b.Fatalf("Close %s, want error to be nil, got %v", fileName, err)
			}
		}
	})
}
