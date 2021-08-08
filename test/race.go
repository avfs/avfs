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
	sfs.RunTests(t, UsrTest,
		sfs.TestRaceDir,
		sfs.TestRaceFile,
		sfs.TestRaceMkdirRemoveAll)
}

// TestRaceDir tests data race conditions for some directory functions.
func (sfs *SuiteFS) TestRaceDir(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	path := vfs.Join(testDir, "mkdDirNew")

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
func (sfs *SuiteFS) TestRaceFile(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	t.Run("RaceCreate", func(t *testing.T) {
		sfs.RaceFunc(t, "Create", RaceAllOk, func() error {
			newFile := vfs.Join(testDir, "newFile")
			f, err := vfs.Create(newFile)
			if err == nil {
				defer f.Close()
			}

			return err
		})
	})

	t.Run("RaceOpenExcl", func(t *testing.T) {
		sfs.RaceFunc(t, "Open Excl", RaceOneOk, func() error {
			newFile := vfs.Join(testDir, "newFileExcl")
			f, err := vfs.OpenFile(newFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
			if err == nil {
				defer f.Close()
			}

			return err
		})
	})

	t.Run("RaceOpenReadOnly", func(t *testing.T) {
		roFile := sfs.EmptyFile(t, testDir)

		sfs.RaceFunc(t, "Open Read Only", RaceAllOk, func() error {
			f, err := vfs.Open(roFile)
			if err == nil {
				defer f.Close()
			}

			return err
		})
	})

	t.Run("RaceFileClose", func(t *testing.T) {
		path := sfs.EmptyFile(t, testDir)

		f, err := vfs.Open(path)
		if err != nil {
			t.Fatalf("Open %s : want error to be nil, got %v", path, err)
		}

		sfs.RaceFunc(t, "Close", RaceOneOk, f.Close)
	})
}

// TestRaceMkdirRemoveAll test data race conditions for MkdirAll and RemoveAll.
func (sfs *SuiteFS) TestRaceMkdirRemoveAll(t *testing.T, testDir string) {
	vfs := sfs.vfsSetup

	path := vfs.Join(testDir, "new/path/to/test")

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

// RaceFunc tests data race conditions by running simultaneously all testFuncs in SuiteFS.maxRace goroutines
// and expecting a result rr.
func (sfs *SuiteFS) RaceFunc(t *testing.T, name string, rr RaceResult, testFuncs ...func() error) {
	var (
		wgSetup    sync.WaitGroup
		wgTeardown sync.WaitGroup
		starter    sync.RWMutex
		wantOk     uint32
		gotOk      uint32
		wantErr    uint32
		gotErr     uint32
	)

	t.Run("Race_"+name, func(t *testing.T) {
		maxGo := sfs.maxRace * len(testFuncs)

		wgSetup.Add(maxGo)
		wgTeardown.Add(maxGo)

		starter.Lock()

		for i := 0; i < sfs.maxRace; i++ {
			for _, testFunc := range testFuncs {
				go func(f func() error) {
					defer func() {
						starter.RUnlock()
						wgTeardown.Done()
					}()

					wgSetup.Done()
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

		// All goroutines wait for the Starter lock.
		wgSetup.Wait()

		// All goroutines execute the testFuncs.
		starter.Unlock()

		// Wait for all goroutines to stop.
		wgTeardown.Wait()

		switch rr {
		case RaceNoneOk:
			wantOk = 0
		case RaceOneOk:
			wantOk = 1
		case RaceAllOk:
			wantOk = uint32(maxGo)
		case RaceUndefined:
			t.Logf("Race %s : ok = %d, error = %d", name, gotOk, gotErr)

			return
		}

		wantErr = uint32(maxGo) - wantOk

		if gotOk != wantOk {
			t.Errorf("Race %s : want number of responses without error to be %d, got %d ", name, wantOk, gotOk)
		}

		if gotErr != wantErr {
			t.Errorf("Race %s : want number of responses with errors to be %d, got %d", name, wantErr, gotErr)
		}
	})
}
