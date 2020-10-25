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

// +build !datarace

package fsutil_test

import (
	"bytes"
	"crypto/sha512"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/fs/osfs"
	"github.com/avfs/avfs/fsutil"
)

func TestHashFile(t *testing.T) {
	vfs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	rtr, err := fsutil.NewRndTree(vfs, fsutil.RndTreeParams{
		MinDepth: 1, MaxDepth: 1,
		MinName: 32, MaxName: 32,
		MinFiles: 100, MaxFiles: 100,
		MinFileLen: 16, MaxFileLen: 100 * 1024,
	})
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	rootDir, err := vfs.TempDir("", avfs.Avfs)
	if err != nil {
		t.Fatalf("TempDir : want error to be nil, got %v", err)
	}

	err = rtr.CreateTree(rootDir)
	if err != nil {
		t.Fatalf("CreateTree : want error to be nil, got %v", err)
	}

	defer vfs.RemoveAll(rootDir) //nolint:errcheck

	h := sha512.New()

	for _, fileName := range rtr.Files {
		content, err := vfs.ReadFile(fileName)
		if err != nil {
			t.Fatalf("ReadFile %s : want error to be nil, got %v", fileName, err)
		}

		h.Reset()

		_, err = h.Write(content)
		if err != nil {
			t.Errorf("Write hash : want error to be nil, got %v", err)
		}

		wantSum := h.Sum(nil)

		gotSum, err := fsutil.HashFile(vfs, fileName, h)
		if err != nil {
			t.Errorf("HashFile %s : want error to be nil, got %v", fileName, err)
		}

		if !bytes.Equal(wantSum, gotSum) {
			t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
		}
	}
}

// TestCopyFile copies random files from an OsFs file system to a MemFs file system.
func TestCopyFile(t *testing.T) {
	srcFs, err := osfs.New()
	if err != nil {
		t.Fatalf("osfs.New : want error to be nil, got %v", err)
	}

	rtr, err := fsutil.NewRndTree(srcFs, fsutil.RndTreeParams{
		MinDepth: 1, MaxDepth: 1,
		MinName: 32, MaxName: 32,
		MinFiles: 512, MaxFiles: 512,
		MinFileLen: 0, MaxFileLen: 100 * 1024,
	})
	if err != nil {
		t.Fatalf("NewRndTree : want error to be nil, got %v", err)
	}

	srcDir, err := srcFs.TempDir("", avfs.Avfs)
	if err != nil {
		t.Fatalf("TempDir : want error to be nil, got %v", err)
	}

	defer srcFs.RemoveAll(srcDir) //nolint:errcheck

	err = rtr.CreateTree(srcDir)
	if err != nil {
		t.Fatalf("CreateTree : want error to be nil, got %v", err)
	}

	dstFs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	h := sha512.New()

	t.Run("CopyFile_WithHashSum", func(t *testing.T) {
		dstDir, err := dstFs.TempDir("", avfs.Avfs)
		if err != nil {
			t.Fatalf("TempDir : want error to be nil, got %v", err)
		}

		defer dstFs.RemoveAll(dstDir) //nolint:errcheck

		for _, srcPath := range rtr.Files {
			fileName := srcFs.Base(srcPath)
			dstPath := fsutil.Join(dstDir, fileName)

			wantSum, err := fsutil.CopyFile(dstFs, srcFs, dstPath, srcPath, h)
			if err != nil {
				t.Errorf("CopyFile (%s)%s, (%s)%s : want error to be nil, got %v",
					dstFs.Type(), dstPath, srcFs.Type(), srcPath, err)
			}

			gotSum, err := fsutil.HashFile(dstFs, dstPath, h)
			if err != nil {
				t.Errorf("HashFile (%s)%s : want error to be nil, got %v", dstFs.Type(), dstPath, err)
			}

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})

	t.Run("CopyFile", func(t *testing.T) {
		dstDir, err := dstFs.TempDir("", avfs.Avfs)
		if err != nil {
			t.Fatalf("TempDir : want error to be nil, got %v", err)
		}

		for _, srcPath := range rtr.Files {
			fileName := srcFs.Base(srcPath)
			dstPath := fsutil.Join(dstDir, fileName)

			wantSum, err := fsutil.CopyFile(dstFs, srcFs, dstPath, srcPath, nil)
			if err != nil {
				t.Errorf("CopyFile (%s)%s, (%s)%s : want error to be nil, got %v",
					dstFs.Type(), dstPath, srcFs.Type(), srcPath, err)
			}

			if wantSum != nil {
				t.Fatalf("CopyFile (%s)%s, (%s)%s) : want error to be nil, got %v",
					dstFs.Type(), dstPath, srcFs.Type(), srcPath, err)
			}

			wantSum, err = fsutil.HashFile(srcFs, srcPath, h)
			if err != nil {
				t.Errorf("HashFile (%s)%s : want error to be nil, got %v", srcFs.Type(), srcPath, err)
			}

			gotSum, err := fsutil.HashFile(dstFs, dstPath, h)
			if err != nil {
				t.Errorf("HashFile (%s)%s : want error to be nil, got %v", dstFs.Type(), dstPath, err)
			}

			if !bytes.Equal(wantSum, gotSum) {
				t.Errorf("HashFile %s : \nwant : %x\ngot  : %x", fileName, wantSum, gotSum)
			}
		}
	})
}
