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

// TestRace tests data race conditions.
func (sfs *SuiteFS) TestRace(t *testing.T) {
	sfs.TestRaceDir(t)
	sfs.TestRaceFile(t)
	sfs.RaceMkdirRemoveAll(t)
}

// TestRaceDir tests data race conditions for some directory functions.
func (sfs *SuiteFS) TestRaceDir(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	path := vfs.Join(rootDir, "mkdDirNew")

	sfs.RaceFunc(t, "Mkdir", RaceOneOk, func() error {
		return vfs.Mkdir(path, avfs.DefaultDirPerm)
	})

	sfs.RaceFunc(t, "Remove", RaceOneOk, func() error {
		return vfs.Remove(path)
	})

	sfs.RaceFunc(t, "MkdirAll", RaceAllOk, func() error {
		return vfs.MkdirAll(path, avfs.DefaultDirPerm)
	})

	sfs.RaceFunc(t, "RemoveAll", RaceAllOk, func() error {
		return vfs.RemoveAll(path)
	})
}

// TestRaceFile tests data race conditions for some file functions.
func (sfs *SuiteFS) TestRaceFile(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	t.Run("RaceCreate", func(t *testing.T) {
		sfs.RaceFunc(t, "Create", RaceAllOk, func() error {
			newFile := vfs.Join(rootDir, "newFile")
			f, err := vfs.Create(newFile)
			if err == nil {
				defer f.Close()
			}

			return err
		})
	})

	t.Run("RaceOpenExcl", func(t *testing.T) {
		sfs.RaceFunc(t, "Open Excl", RaceOneOk, func() error {
			newFile := vfs.Join(rootDir, "newFileExcl")
			f, err := vfs.OpenFile(newFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
			if err == nil {
				defer f.Close()
			}

			return err
		})
	})

	t.Run("RaceOpenReadOnly", func(t *testing.T) {
		roFile := sfs.CreateEmptyFile(t)

		sfs.RaceFunc(t, "Open Read Only", RaceAllOk, func() error {
			f, err := vfs.Open(roFile)
			if err == nil {
				defer f.Close()
			}

			return err
		})
	})

	t.Run("RaceFileClose", func(t *testing.T) {
		f := sfs.CreateAndOpenFile(t)

		sfs.RaceFunc(t, "Close", RaceOneOk, f.Close)
	})
}

// RaceMkdirRemoveAll test data race conditions for MkdirAll and RemoveAll.
func (sfs *SuiteFS) RaceMkdirRemoveAll(t *testing.T) {
	rootDir, removeDir := sfs.CreateRootDir(t, UsrTest)
	defer removeDir()

	vfs := sfs.vfsWrite

	path := vfs.Join(rootDir, "new/path/to/test")

	sfs.RaceFunc(t, "Complex", RaceUndefined, func() error {
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
func (sfs *SuiteFS) RaceFunc(t *testing.T, name string, rr RaceResult, testFuncs ...func() error) {
	var (
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
