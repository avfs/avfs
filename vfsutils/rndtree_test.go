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

package vfsutils_test

import (
	"strconv"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/vfsutils"
)

var (
	ErrDepthOutOfRange    = vfsutils.ErrOutOfRange("depth")
	ErrNameOutOfRange     = vfsutils.ErrOutOfRange("name")
	ErrDirsOutOfRange     = vfsutils.ErrOutOfRange("dirs")
	ErrFilesOutOfRange    = vfsutils.ErrOutOfRange("files")
	ErrFileLenOutOfRange  = vfsutils.ErrOutOfRange("file length")
	ErrSymlinksOutOfRange = vfsutils.ErrOutOfRange("symbolic links")
)

func TestErrOutOfRange(t *testing.T) {
	parameter := "Some"
	wantErrStr := parameter + " parameter out of range"

	err := vfsutils.ErrOutOfRange(parameter)
	if err.Error() != wantErrStr {
		t.Errorf("ErrOutOfRange : want error to be %s, got %s", wantErrStr, err.Error())
	}
}

func TestRndTree(t *testing.T) {
	vfs, err := memfs.New(memfs.WithMainDirs())
	if err != nil {
		t.Errorf("New : want error to be nil, got %v", err)
	}

	tmpDir, err := vfs.TempDir("", avfs.Avfs)
	if err != nil {
		t.Fatalf("TempDir : want error to be nil, got %v", err)
	}

	defer vfs.RemoveAll(tmpDir) //nolint:errcheck

	t.Run("RndTreeMain", func(t *testing.T) {
		rtrTests := []struct {
			params  vfsutils.RndTreeParams
			wantErr error
		}{
			{params: vfsutils.RndTreeParams{MinDepth: 0, MaxDepth: 0}, wantErr: ErrDepthOutOfRange},
			{params: vfsutils.RndTreeParams{MinDepth: 1, MaxDepth: 0}, wantErr: ErrDepthOutOfRange},
			{params: vfsutils.RndTreeParams{MinDepth: 1, MaxDepth: 1, MinName: 0, MaxName: 0}, wantErr: ErrNameOutOfRange},
			{params: vfsutils.RndTreeParams{MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 0}, wantErr: ErrNameOutOfRange},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinDirs: -1, MaxDirs: 0,
			}, wantErr: ErrDirsOutOfRange},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinDirs: 1, MaxDirs: 0,
			}, wantErr: ErrDirsOutOfRange},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinFiles: -1, MaxFiles: 0,
			}, wantErr: ErrFilesOutOfRange},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinFiles: 1, MaxFiles: 0,
			}, wantErr: ErrFilesOutOfRange},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinFileLen: -1, MaxFileLen: 0,
			}, wantErr: ErrFileLenOutOfRange},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinFileLen: 1, MaxFileLen: 0,
			}, wantErr: ErrFileLenOutOfRange},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinSymlinks: -1, MaxSymlinks: 0,
			}, wantErr: ErrSymlinksOutOfRange},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 1, MaxName: 1,
				MinSymlinks: 1, MaxSymlinks: 0,
			}, wantErr: ErrSymlinksOutOfRange},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 1, MinName: 10, MaxName: 20,
				MinDirs: 5, MaxDirs: 10, MinFiles: 5, MaxFiles: 10, MinFileLen: 5, MaxFileLen: 10,
				MinSymlinks: 5, MaxSymlinks: 10,
			}, wantErr: nil},
			{params: vfsutils.RndTreeParams{
				MinDepth: 1, MaxDepth: 3, MinName: 10, MaxName: 10,
				MinDirs: 3, MaxDirs: 3, MinFiles: 3, MaxFiles: 3, MinFileLen: 3, MaxFileLen: 3,
				MinSymlinks: 3, MaxSymlinks: 3,
			}, wantErr: nil},
		}

		for i, rtrTest := range rtrTests {
			rtr, err := vfsutils.NewRndTree(vfs, rtrTest.params)

			if rtrTest.wantErr == nil {
				if err != nil {
					t.Errorf("NewRndTree %d: want error to be nil, got %v", i, err)

					continue
				}
			} else {
				if err == nil {
					t.Errorf("NewRndTree %d : want error to be %v, got nil", i, rtrTest.wantErr)
				} else if rtrTest.wantErr != err {
					t.Errorf("NewRndTree %d : want error to be %v, got %v", i, rtrTest.wantErr, err)
				}

				continue
			}

			path := vfs.Join(tmpDir, "Main", strconv.Itoa(i))

			err = vfs.MkdirAll(path, avfs.DefaultDirPerm)
			if err != nil {
				t.Fatalf("MkdirAll %s : want error to be nil, got %v", path, err)
			}

			err = rtr.CreateTree(path)
			if err != nil {
				t.Errorf("CreateTree : want error to be nil, got %v", err)
			}

			if rtr.MaxDepth == 0 {
				ld := len(rtr.Dirs)
				if ld < rtr.MinDirs || ld > rtr.MaxDirs {
					t.Errorf("CreateTree : want dirs number to to be between %d and %d, got %d",
						rtr.MinDirs, rtr.MaxDirs, ld)
				}

				lf := len(rtr.Files)
				if lf < rtr.MinFiles || lf > rtr.MaxFiles {
					t.Errorf("CreateTree : want files number to to be between %d and %d, got %d",
						rtr.MinFiles, rtr.MaxFiles, lf)
				}

				ls := len(rtr.SymLinks)
				if ls < rtr.MinSymlinks || ls > rtr.MaxSymlinks {
					t.Errorf("CreateTree : want symbolic linls number to to be between %d and %d, got %d",
						rtr.MinSymlinks, rtr.MaxSymlinks, ls)
				}
			}
		}
	})

	t.Run("RndTreeDepth", func(t *testing.T) {
		rtr, err := vfsutils.NewRndTree(vfs, vfsutils.RndTreeParams{
			MinDepth: 3, MaxDepth: 3,
			MinName: 10, MaxName: 10,
			MinDirs: 2, MaxDirs: 2,
			MinFiles: 1, MaxFiles: 1,
		})
		if err != nil {
			t.Errorf("NewRndTree : want error to be nil, got %v", err)
		}

		path := vfs.Join(tmpDir, "Depth")

		err = vfs.MkdirAll(path, avfs.DefaultDirPerm)
		if err != nil {
			t.Fatalf("MkdirAll %s : want error to be nil, got %v", path, err)
		}

		err = rtr.CreateTree(path)
		if err != nil {
			t.Errorf("CreateTree : want error to be nil, got %v", err)
		}

		wantDirs := 14
		if len(rtr.Dirs) != wantDirs {
			t.Errorf("CreateTree : want number of directories to be %d, got %d", wantDirs, len(rtr.Dirs))
		}

		wantFiles := 7
		if len(rtr.Files) != wantFiles {
			t.Errorf("CreateTree : want number of directories to be %d, got %d", wantFiles, len(rtr.Files))
		}
	})
}
