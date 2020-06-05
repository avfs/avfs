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
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/avfs/avfs"
)

// TestRaceMkdir tests race conditions for mkdir function.
func (cf *ConfigFs) TestRaceMkdir() {
	t, rootDir, removeDir := cf.CreateRootDir(UsrTest)
	defer removeDir()

	const maxGo = 10000

	var (
		simCount int32
		simMax   int32
		wg       sync.WaitGroup
	)

	fs := cf.GetFsWrite()

	wg.Add(maxGo)

	for i := 0; i < maxGo; i++ {
		go func(i int) {
			defer wg.Done()

			path := fs.Join(rootDir, strconv.Itoa(i))
			scm := atomic.LoadInt32(&simMax)

			sc := atomic.AddInt32(&simCount, 1)
			if sc > scm {
				atomic.CompareAndSwapInt32(&simMax, scm, sc)
			}

			err := fs.Mkdir(path, avfs.DefaultDirPerm)
			if err != nil {
				if e, ok := err.(*os.PathError); !ok || e.Err != avfs.ErrFileExists {
					t.Errorf("mkdir %s : want error to be nil or %v, got %v", path, avfs.ErrFileExists, err)
				}
			}

			atomic.AddInt32(&simCount, -1)
		}(i)
	}
	wg.Wait()

	t.Logf("max= %d", simMax)
}
