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
func (ts *Suite) TestRace(t *testing.T) {
	vfs := ts.vfsTest

	if vfs.OSType() != avfs.CurrentOSType() {
		t.Skipf("TestRace : Current OSType = %s is different from %s OSType = %s, skipping race tests",
			avfs.CurrentOSType(), vfs.Type(), vfs.OSType())
	}

	ts.RunTests(t, UsrTest,
		ts.RaceCreate,
		ts.RaceCreateTemp,
		ts.RaceFileClose,
		ts.RaceMkdir,
		ts.RaceMkdirAll,
		ts.RaceMkdirTemp,
		ts.RaceOpen,
		ts.RaceOpenFile,
		ts.RaceOpenFileExcl,
		ts.RaceRemove,
		ts.RaceRemoveAll,
		ts.RaceMkdirRemoveAll)
}

// RaceCreate tests data race conditions for Create.
func (ts *Suite) RaceCreate(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	ts.raceFunc(t, RaceAllOk, func() error {
		newFile := vfs.Join(testDir, defaultFile)

		f, err := vfs.Create(newFile)
		if err == nil {
			defer f.Close()
		}

		return err
	})
}

// RaceCreateTemp tests data race conditions for CreateTemp.
func (ts *Suite) RaceCreateTemp(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	var fileNames sync.Map

	ts.raceFunc(t, RaceAllOk, func() error {
		fileName, err := vfs.CreateTemp(testDir, "avfs")

		_, exists := fileNames.LoadOrStore(fileName, nil)
		if exists {
			t.Errorf("file %s already exists", fileName)
		}

		return err
	})
}

// RaceMkdir tests data race conditions for Mkdir.
func (ts *Suite) RaceMkdir(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	path := vfs.Join(testDir, defaultDir)

	ts.raceFunc(t, RaceOneOk, func() error {
		return vfs.Mkdir(path, avfs.DefaultDirPerm)
	})
}

// RaceMkdirAll tests data race conditions for MkdirAll.
func (ts *Suite) RaceMkdirAll(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	path := vfs.Join(testDir, defaultDir)

	ts.raceFunc(t, RaceAllOk, func() error {
		return vfs.MkdirAll(path, avfs.DefaultDirPerm)
	})
}

// RaceMkdirTemp tests data race conditions for MkdirTemp.
func (ts *Suite) RaceMkdirTemp(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	var dirs sync.Map

	ts.raceFunc(t, RaceAllOk, func() error {
		dir, err := vfs.MkdirTemp(testDir, "RaceMkdirTemp")

		_, exists := dirs.LoadOrStore(dir, nil)
		if exists {
			t.Errorf("directory %s already exists", dir)
		}

		return err
	})
}

// RaceOpen tests data race conditions for Open.
func (ts *Suite) RaceOpen(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	roFile := ts.emptyFile(t, testDir)

	ts.raceFunc(t, RaceAllOk, func() error {
		f, err := vfs.OpenFile(roFile, os.O_RDONLY, 0)
		if err == nil {
			defer f.Close()
		}

		return err
	})
}

// RaceOpenFile tests data race conditions for OpenFile.
func (ts *Suite) RaceOpenFile(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	newFile := vfs.Join(testDir, defaultFile)

	ts.raceFunc(t, RaceAllOk, func() error {
		f, err := vfs.OpenFile(newFile, os.O_RDWR|os.O_CREATE, avfs.DefaultFilePerm)
		if err == nil {
			defer f.Close()
		}

		return err
	})
}

// RaceOpenFileExcl tests data race conditions for OpenFile with O_EXCL flag.
func (ts *Suite) RaceOpenFileExcl(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	newFile := vfs.Join(testDir, defaultFile)

	ts.raceFunc(t, RaceOneOk, func() error {
		f, err := vfs.OpenFile(newFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, avfs.DefaultFilePerm)
		if err == nil {
			defer f.Close()
		}

		return err
	})
}

// RaceRemove tests data race conditions for Remove.
func (ts *Suite) RaceRemove(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	path := vfs.Join(testDir, defaultDir)

	ts.createDir(t, path, avfs.DefaultDirPerm)

	ts.raceFunc(t, RaceUndefined, func() error {
		return vfs.Remove(path)
	})
}

// RaceRemoveAll tests data race conditions for RemoveAll.
func (ts *Suite) RaceRemoveAll(t *testing.T, testDir string) {
	vfs := ts.vfsTest
	path := vfs.Join(testDir, defaultDir)

	ts.createDir(t, path, avfs.DefaultDirPerm)

	ts.raceFunc(t, RaceAllOk, func() error {
		return vfs.RemoveAll(path)
	})
}

// RaceFileClose tests data race conditions for File.Close.
func (ts *Suite) RaceFileClose(t *testing.T, testDir string) {
	f, _ := ts.openedEmptyFile(t, testDir)

	ts.raceFunc(t, RaceOneOk, f.Close)
}

// RaceMkdirRemoveAll test data race conditions for MkdirAll and RemoveAll.
func (ts *Suite) RaceMkdirRemoveAll(t *testing.T, testDir string) {
	vfs := ts.vfsTest

	path := vfs.Join(testDir, "new/path/to/test")

	ts.raceFunc(t, RaceUndefined, func() error {
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

// raceFunc tests data race conditions by running simultaneously all testFuncs in Suite.maxRace goroutines
// and expecting a result rr.
func (ts *Suite) raceFunc(t *testing.T, rr RaceResult, testFuncs ...func() error) {
	var (
		wgSetup    sync.WaitGroup
		wgTeardown sync.WaitGroup
		starter    sync.RWMutex
		wantOk     uint32
		gotOk      uint32
		wantErr    uint32
		gotErr     uint32
	)

	maxGo := ts.maxRace * len(testFuncs)

	wgSetup.Add(maxGo)
	wgTeardown.Add(maxGo)

	starter.Lock()

	for range ts.maxRace {
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
