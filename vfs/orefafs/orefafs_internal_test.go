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

package orefafs

import (
	"testing"

	"github.com/avfs/avfs"
)

func TestSplitPath(t *testing.T) {
	var vfs *OrefaFS

	vfsLinux := New(WithOSType(avfs.OsLinux))
	vfsWindows := New(WithOSType(avfs.OsWindows))

	cases := []struct {
		path string
		dir  string
		file string
		ost  avfs.OSType
	}{
		{ost: avfs.OsLinux, path: "/", dir: "", file: ""},
		{ost: avfs.OsLinux, path: "/home", dir: "", file: "home"},
		{ost: avfs.OsLinux, path: "/home/user", dir: "/home", file: "user"},
		{ost: avfs.OsLinux, path: "/usr/lib/xorg", dir: "/usr/lib", file: "xorg"},
		{ost: avfs.OsWindows, path: `C:\`, dir: `C:`, file: ""},
		{ost: avfs.OsWindows, path: `C:\Users`, dir: `C:`, file: "Users"},
		{ost: avfs.OsWindows, path: `C:\Users\Default`, dir: `C:\Users`, file: "Default"},
	}

	for _, c := range cases {
		switch c.ost {
		case avfs.OsWindows:
			vfs = vfsWindows
		default:
			vfs = vfsLinux
		}

		dir, file := vfs.splitPath(c.path)
		if c.dir != dir {
			t.Errorf("splitPath %s : want dir to be %s, got %s", c.path, c.dir, dir)
		}

		if c.file != file {
			t.Errorf("splitPath %s : want file to be %s, got %s", c.path, c.file, file)
		}
	}
}
