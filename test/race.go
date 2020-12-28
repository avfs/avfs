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
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/avfs/avfs"
)

// Race tests data race conditions.
func (sfs *SuiteFS) Race() {
	sfs.RaceDir()
	sfs.RaceFile()
	sfs.RaceMkdirRemoveAll()
}

// RaceDir tests data race conditions for some directory functions.
func (sfs *SuiteFS) RaceDir() {
	_, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	path := vfs.Join(rootDir, "mkdDirNew")

	sfs.RaceFunc("Mkdir", RaceOneOk, func() error {
		return vfs.Mkdir(path, avfs.DefaultDirPerm)
	})

	sfs.RaceFunc("Remove", RaceOneOk, func() error {
		return vfs.Remove(path)
	})

	sfs.RaceFunc("MkdirAll", RaceAllOk, func() error {
		return vfs.MkdirAll(path, avfs.DefaultDirPerm)
	})

	sfs.RaceFunc("RemoveAll", RaceAllOk, func() error {
		return vfs.RemoveAll(path)
	})
}

// RaceFile tests data race conditions for some file functions.
func (sfs *SuiteFS) RaceFile() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	sfs.RaceFunc("Create", RaceAllOk, func() error {
		newFile := vfs.Join(rootDir, "newFile")
		f, err := vfs.Create(newFile)
		if err == nil {
			defer f.Close()
		}

		return err
	})

	sfs.RaceFunc("Open Excl", RaceOneOk, func() error {
		newFile := vfs.Join(rootDir, "newFileExcl")
		f, err := vfs.OpenFile(newFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		if err == nil {
			defer f.Close()
		}

		return err
	})

	func() {
		roFile := vfs.Join(rootDir, "roFile")

		err := vfs.WriteFile(roFile, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want err to be nil, got %v", err)
		}

		sfs.RaceFunc("Open Read Only", RaceAllOk, func() error {
			f, err := vfs.Open(roFile)
			if err == nil {
				defer f.Close()
			}

			return err
		})
	}()

	func() {
		newFile := vfs.Join(rootDir, "newFileClose")

		f, err := vfs.Create(newFile)
		if err != nil {
			t.Fatalf("Create : want err to be nil, got %v", err)
		}

		sfs.RaceFunc("Close", RaceOneOk, f.Close)
	}()
}

// RaceMkdirRemoveAll test data race conditions for MkdirAll and RemoveAll.
func (sfs *SuiteFS) RaceMkdirRemoveAll() {
	_, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	vfs := sfs.GetFsWrite()

	path := vfs.Join(rootDir, "new/path/to/test")

	sfs.RaceFunc("Complex", RaceUndefined, func() error {
		return vfs.MkdirAll(path, avfs.DefaultDirPerm)
	}, func() error {
		return vfs.RemoveAll(path)
	})
}

// RaceResult defines the type of result expected from a race test.
type RaceResult uint8

const (
	// RaceNoneOk expects that all the results will return an error.
	RaceNoneOk RaceResult = iota + 1

	// RaceOneOk expects that only one result will be without error.
	RaceOneOk

	// RaceAllOk expects that all results will be without error.
	RaceAllOk

	// RaceUndefined does not expect anything (unpredictable results).
	RaceUndefined
)

// RaceFunc tests data race conditions by running simultaneously all testFuncs in cf.maxRace goroutines
// and expecting a result rr.
func (sfs *SuiteFS) RaceFunc(name string, rr RaceResult, testFuncs ...func() error) {
	var (
		t       = sfs.t
		wg      sync.WaitGroup
		starter sync.RWMutex
		wantOk  uint32
		gotOk   uint32
		wantErr uint32
		gotErr  uint32
	)

	t.Run("Race_"+name, func(t *testing.T) {
		wg.Add(sfs.maxRace * len(testFuncs))
		starter.Lock()

		for i := 0; i < sfs.maxRace; i++ {
			for _, testFunc := range testFuncs {
				go func(f func() error) {
					defer func() {
						starter.RUnlock()
						wg.Done()
					}()

					starter.RLock()

					err := f()
					if err != nil {
						atomic.AddUint32(&gotErr, 1)

						return
					}

					atomic.AddUint32(&gotOk, 1)
				}(testFunc)
			}
		}

		starter.Unlock()
		wg.Wait()

		switch rr {
		case RaceNoneOk:
			wantOk = 0
		case RaceOneOk:
			wantOk = 1
		case RaceAllOk:
			wantOk = uint32(sfs.maxRace)
		case RaceUndefined:
			t.Logf("Race %s : ok = %d, error = %d", name, gotOk, gotErr)

			return
		}

		wantErr = uint32(sfs.maxRace) - wantOk

		if gotOk != wantOk {
			t.Errorf("Race %s : want number of responses without error to be %d, got %d ", name, wantOk, gotOk)
		}

		if gotErr != wantErr {
			t.Errorf("Race %s : want number of responses with errors to be %d, got %d", name, wantErr, gotErr)
		}
	})
}
