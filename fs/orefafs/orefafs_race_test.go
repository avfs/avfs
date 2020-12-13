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

//+build datarace

package orefafs_test

import (
	"testing"

	"github.com/avfs/avfs/fs/orefafs"
	"github.com/avfs/avfs/test"
)

func TestRacOrefaFs(t *testing.T) {
	vfs, err := orefafs.New(orefafs.WithMainDirs())
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(t, vfs)

	sfs.SuiteRace()
}
