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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/avfs/avfs"
)

// SuiteRace tests data race conditions for some functions.
func (cf *ConfigFs) SuiteRace() {
	_, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	fs := cf.GetFsWrite()

	path := fs.Join(rootDir, "mkdDirNew")

	cf.SuiteRaceFunc("Mkdir", RaceOneOk, func() error {
		return fs.Mkdir(path, avfs.DefaultDirPerm)
	})

	cf.SuiteRaceFunc("Remove", RaceOneOk, func() error {
		return fs.Remove(path)
	})

	cf.SuiteRaceFunc("MkdirAll", RaceAllOk, func() error {
		return fs.MkdirAll(path, avfs.DefaultDirPerm)
	})

	cf.SuiteRaceFunc("Lstat Ok", RaceAllOk, func() error {
		_, err := fs.Lstat(path)
		return err
	})

	cf.SuiteRaceFunc("Stat Ok", RaceAllOk, func() error {
		_, err := fs.Stat(path)
		return err
	})

	cf.SuiteRaceFunc("RemoveAll", RaceAllOk, func() error {
		return fs.RemoveAll(path)
	})

	cf.SuiteRaceFunc("Lstat Error", RaceNoneOk, func() error {
		_, err := fs.Lstat(path)
		return err
	})

	cf.SuiteRaceFunc("Stat Error", RaceNoneOk, func() error {
		_, err := fs.Stat(path)
		return err
	})
}

// RaceResult defines the type of result expected from a race test.
type RaceResult uint8

const (
	// RaceNoneOk expects that all the results will return an error.
	RaceNoneOk RaceResult = iota

	// RaceOneOk expects that only one result will be without error.
	RaceOneOk

	// RaceAllOk expects that all results will be without error.
	RaceAllOk
)

// SuiteRaceFunc tests data race conditions on a function f expecting a result rr.
func (cf *ConfigFs) SuiteRaceFunc(name string, rr RaceResult, f func() error) {
	var (
		t       = cf.t
		wg      sync.WaitGroup
		starter sync.RWMutex
		wantOk  int32
		gotOk   int32
		wantErr int32
		gotErr  int32
	)

	t.Run("Race_"+name, func(t *testing.T) {
		wg.Add(cf.maxRace)
		starter.Lock()

		for i := 0; i < cf.maxRace; i++ {
			go func() {
				defer func() {
					starter.RUnlock()
					wg.Done()
				}()

				starter.RLock()

				err := f()
				if err != nil {
					atomic.AddInt32(&gotErr, 1)

					return
				}

				atomic.AddInt32(&gotOk, 1)
			}()
		}

		starter.Unlock()
		wg.Wait()

		switch rr {
		case RaceNoneOk:
			wantOk = 0
		case RaceOneOk:
			wantOk = 1
		case RaceAllOk:
			wantOk = int32(cf.maxRace)
		}

		wantErr = int32(cf.maxRace) - wantOk

		if gotOk != wantOk {
			t.Errorf("Race %s : want number of responses without error to be %d, got %d ", name, wantOk, gotOk)
		}

		if gotErr != wantErr {
			t.Errorf("Race %s : want number of responses with errors to be %d, got %d", name, wantErr, gotErr)
		}
	})
}
