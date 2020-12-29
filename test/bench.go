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
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfsutils"
)

// BenchmarkCreate is a simple benchmark to create a random tree.
func BenchmarkCreate(b *testing.B, vfs avfs.VFS) {
	rtr, err := vfsutils.NewRndTree(vfs, vfsutils.RndTreeParams{
		MinName: 32, MaxName: 32,
		MinDepth: 4, MaxDepth: 4,
		MinDirs: 2, MaxDirs: 2,
		MinFiles: 5, MaxFiles: 5,
		MinFileLen: 128, MaxFileLen: 128,
	})
	if err != nil {
		b.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		rand.Seed(42)

		rootDir, err := vfs.TempDir("", avfs.Avfs)
		if err != nil {
			b.Fatalf("TempDir : want error to be nil, got %v", err)
		}

		err = rtr.CreateTree(rootDir)
		if err != nil {
			b.Fatalf("CreateTree : want error to be nil, got %v", err)
		}

		err = vfs.RemoveAll(rootDir)
		if err != nil {
			b.Fatalf("RemoveAll : want error to be nil, got %v", err)
		}
	}
}
