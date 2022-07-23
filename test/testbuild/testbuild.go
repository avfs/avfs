//
//  Copyright 2022 The AVFS authors
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

package main

import (
	"log"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"
)

// This code is used by gox to generate an executable for all operating systems (see mage/magefile.go).
// It should use all functions that depend on OS specific syscalls to make sure every system can be built.
func main() {
	vfs := osfs.New()

	tmpDir, err := vfs.MkdirTemp("", "gox")
	if err != nil {
		log.Fatal(err)
	}

	tmpFile := vfs.Join(tmpDir, "file")

	err = vfs.WriteFile(tmpFile, nil, avfs.DefaultFilePerm)
	if err != nil {
		log.Fatal(err)
	}

	info, err := vfs.Stat(tmpFile)
	if err != nil {
		log.Fatal(err)
	}

	sst := vfs.ToSysStat(info)

	log.Printf("gid = %d\nuid = %d\nnlink = %d\n", sst.Gid(), sst.Uid(), sst.Nlink())
}
