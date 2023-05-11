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
	vfs := sfs.vfsTest

	if vfs.OSType() != avfs.CurrentOSType() {
		t.Skipf("TestRace : Current OSType = %s is different from %s OSType = %s, skipping race tests",
			avfs.CurrentOSType(), vfs.Type(), vfs.OSType())
	}

	sfs.RunTests(t, UsrTest,
		sfs.RaceCreate,
		sfs.RaceCreateTemp,
		sfs.RaceFileClose,
		sfs.RaceMkdir,
		sfs.RaceMkdirAll,
		sfs.RaceMkdirTemp,
		sfs.RaceOpen,
		sfs.RaceOpenFile,
		sfs.RaceOpenFileExcl,
		sfs.RaceRemove,
		sfs.RaceRemoveAll,

		sfs.RaceMkdirRemoveAll)
}

// RaceCreate tests data race conditions for Create.
func (sfs *SuiteFS) RaceCreate(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	sfs.raceFunc(t, RaceAllOk, func() error {
		newFile := vfs.Join(testDir, defaultFile)

		f, err := vfs.Create(newFile)
		if err == nil {
			defer f.Close()
		}

		return err
	})
}

// RaceCreateTemp tests data race conditions for CreateTemp.
func (sfs *SuiteFS) RaceCreateTemp(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	var fileNames sync.Map

	sfs.raceFunc(t, RaceAllOk, func() error {
		fileName, err := vfs.CreateTemp(testDir, "avfs")

		_, exists := fileNames.LoadOrStore(fileName, nil)
		if exists {
			t.Errorf("file %s already exists", fileName)
		}

		return err
	})
}

// RaceMkdir tests data race conditions for Mkdir.
func (sfs *SuiteFS) RaceMkdir(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	path := vfs.Join(testDir, defaultDir)

	sfs.raceFunc(t, RaceOneOk, func() error {
		return vfs.Mkdir(path, avfs.DefaultDirPerm)
	})
}

// RaceMkdirAll tests data race conditions for MkdirAll.
func (sfs *SuiteFS) RaceMkdirAll(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	path := vfs.Join(testDir, defaultDir)

	sfs.raceFunc(t, RaceAllOk, func() error {
		return vfs.MkdirAll(path, avfs.DefaultDirPerm)
	})
}

// RaceMkdirTemp tests data race conditions for MkdirTemp.
func (sfs *SuiteFS) RaceMkdirTemp(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	var dirs sync.Map

	sfs.raceFunc(t, RaceAllOk, func() error {
		dir, err := vfs.MkdirTemp(testDir, "RaceMkdirTemp")

		_, exists := dirs.LoadOrStore(dir, nil)
		if exists {
			t.Errorf("directory %s already exists", dir)
		}

		return err
	})
}

// RaceOpen tests data race conditions for Open.
func (sfs *SuiteFS) RaceOpen(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	roFile := sfs.emptyFile(t, testDir)

	sfs.raceFunc(t, RaceAllOk, func() error {
		f, err := vfs.OpenFile(roFile, os.O_RDONLY, 0)
		if err == nil {
			defer f.Close()
		}

		return err
	})
}

// RaceOpenFile tests data race conditions for OpenFile.
func (sfs *SuiteFS) RaceOpenFile(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	newFile := vfs.Join(testDir, defaultFile)

	sfs.raceFunc(t, RaceAllOk, func() error {
		f, err := vfs.OpenFile(newFile, os.O_RDWR|os.O_CREATE, avfs.DefaultFilePerm)
		if err == nil {
			defer f.Close()
		}

		return err
	})
}

// RaceOpenFileExcl tests data race conditions for OpenFile with O_EXCL flag.
func (sfs *SuiteFS) RaceOpenFileExcl(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	newFile := vfs.Join(testDir, defaultFile)

	sfs.raceFunc(t, RaceOneOk, func() error {
		f, err := vfs.OpenFile(newFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		if err == nil {
			defer f.Close()
		}

		return err
	})
}

// RaceRemove tests data race conditions for Remove.
func (sfs *SuiteFS) RaceRemove(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	path := vfs.Join(testDir, defaultDir)

	sfs.createDir(t, path, avfs.DefaultDirPerm)

	sfs.raceFunc(t, RaceUndefined, func() error {
		return vfs.Remove(path)
	})
}

// RaceRemoveAll tests data race conditions for RemoveAll.
func (sfs *SuiteFS) RaceRemoveAll(t *testing.T, testDir string) {
	vfs := sfs.vfsTest
	path := vfs.Join(testDir, defaultDir)

	sfs.createDir(t, path, avfs.DefaultDirPerm)

	sfs.raceFunc(t, RaceAllOk, func() error {
		return vfs.RemoveAll(path)
	})
}

// RaceFileClose tests data race conditions for File.Close.
func (sfs *SuiteFS) RaceFileClose(t *testing.T, testDir string) {
	f, _ := sfs.openedEmptyFile(t, testDir)

	sfs.raceFunc(t, RaceOneOk, f.Close)
}

// RaceMkdirRemoveAll test data race conditions for MkdirAll and RemoveAll.
func (sfs *SuiteFS) RaceMkdirRemoveAll(t *testing.T, testDir string) {
	vfs := sfs.vfsTest

	path := vfs.Join(testDir, "new/path/to/test")

	sfs.raceFunc(t, RaceUndefined, func() error {
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

// raceFunc tests data race conditions by running simultaneously all testFuncs in SuiteFS.maxRace goroutines
// and expecting a result rr.
func (sfs *SuiteFS) raceFunc(t *testing.T, rr RaceResult, testFuncs ...func() error) {
	var (
		wgSetup    sync.WaitGroup
		wgTeardown sync.WaitGroup
		starter    sync.RWMutex
		wantOk     uint32
		gotOk      uint32
		wantErr    uint32
		gotErr     uint32
	)

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
		t.Logf("ok = %d, error = %d", gotOk, gotErr)

		return
	}

	wantErr = uint32(maxGo) - wantOk

	if gotOk != wantOk {
		t.Errorf("want number of responses without error to be %d, got %d ", wantOk, gotOk)
	}

	if gotErr != wantErr {
		t.Errorf("want number of responses with errors to be %d, got %d", wantErr, gotErr)
	}
}
