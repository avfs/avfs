//
//  Copyright 2021 The AVFS authors
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

package avfs_test

import (
	"testing"

	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/orefafs"
)

func TestUtilsMemFS(t *testing.T) {
	vfs := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestAll(t)
}

func TestUtilsOrefaFS(t *testing.T) {
	vfs := orefafs.New(orefafs.WithMainDirs())

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestAll(t)
}
