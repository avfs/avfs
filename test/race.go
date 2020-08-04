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

// SuiteRace tests data race conditions.
func (sfs *SuiteFs) SuiteRace() {
	sfs.SuiteRaceDir()
	sfs.SuiteRaceFile()
	sfs.SuiteRaceMkdirRemoveAll()
}

// SuiteRaceDir tests data race conditions for some directory functions.
func (sfs *SuiteFs) SuiteRaceDir() {
	_, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()

	path := fs.Join(rootDir, "mkdDirNew")

	sfs.SuiteRaceFunc("Mkdir", RaceOneOk, func() error {
		return fs.Mkdir(path, avfs.DefaultDirPerm)
	})

	sfs.SuiteRaceFunc("Remove", RaceOneOk, func() error {
		return fs.Remove(path)
	})

	sfs.SuiteRaceFunc("MkdirAll", RaceAllOk, func() error {
		return fs.MkdirAll(path, avfs.DefaultDirPerm)
	})

	sfs.SuiteRaceFunc("RemoveAll", RaceAllOk, func() error {
		return fs.RemoveAll(path)
	})
}

// SuiteRaceFile tests data race conditions for some file functions.
func (sfs *SuiteFs) SuiteRaceFile() {
	t, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()

	sfs.SuiteRaceFunc("Create", RaceAllOk, func() error {
		newFile := fs.Join(rootDir, "newFile")
		f, err := fs.Create(newFile)
		if err == nil {
			defer f.Close()
		}

		return err
	})

	sfs.SuiteRaceFunc("Open Excl", RaceOneOk, func() error {
		newFile := fs.Join(rootDir, "newFileExcl")
		f, err := fs.OpenFile(newFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		if err == nil {
			defer f.Close()
		}

		return err
	})

	func() {
		roFile := fs.Join(rootDir, "roFile")

		err := fs.WriteFile(roFile, nil, avfs.DefaultFilePerm)
		if err != nil {
			t.Fatalf("WriteFile : want err to be nil, got %v", err)
		}

		sfs.SuiteRaceFunc("Open Read Only", RaceAllOk, func() error {
			f, err := fs.Open(roFile)
			if err == nil {
				defer f.Close()
			}

			return err
		})
	}()

	func() {
		newFile := fs.Join(rootDir, "newFileClose")

		f, err := fs.Create(newFile)
		if err != nil {
			t.Fatalf("Create : want err to be nil, got %v", err)
		}

		sfs.SuiteRaceFunc("Close", RaceOneOk, f.Close)
	}()
}

// SuiteRaceMkdirRemoveAll test data race conditions for MkdirAll and RemoveAll.
func (sfs *SuiteFs) SuiteRaceMkdirRemoveAll() {
	_, rootDir, removeDir := sfs.CreateRootDir(UsrTest)
	defer removeDir()

	fs := sfs.GetFsWrite()

	path := fs.Join(rootDir, "new/path/to/test")

	sfs.SuiteRaceFunc("Complex", RaceUndefined, func() error {
		return fs.MkdirAll(path, avfs.DefaultDirPerm)
	}, func() error {
		return fs.RemoveAll(path)
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

// SuiteRaceFunc tests data race conditions by running simultaneously all testFuncs in cf.maxRace goroutines
// and expecting a result rr.
func (sfs *SuiteFs) SuiteRaceFunc(name string, rr RaceResult, testFuncs ...func() error) {
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
