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

package avfs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
)

// TestPathIterator tests PathIterator methods.
func TestPathIterator(t *testing.T) {
	vfs := memfs.New()

	t.Run("PathIterator", func(t *testing.T) {
		cases := []struct {
			path   string
			parts  []string
			osType avfs.OSType
		}{
			{path: `C:\`, parts: nil, osType: avfs.OsWindows},
			{path: `C:\Users`, parts: []string{"Users"}, osType: avfs.OsWindows},
			{path: `c:\नमस्ते\दुनिया`, parts: []string{"नमस्ते", "दुनिया"}, osType: avfs.OsWindows},

			{path: "/", parts: nil, osType: avfs.OsLinux},
			{path: "/a", parts: []string{"a"}, osType: avfs.OsLinux},
			{path: "/b/c/d", parts: []string{"b", "c", "d"}, osType: avfs.OsLinux},
			{path: "/नमस्ते/दुनिया", parts: []string{"नमस्ते", "दुनिया"}, osType: avfs.OsLinux},
		}

		for _, c := range cases {
			if c.osType != avfs.CurrentOSType() {
				continue
			}

			pi := avfs.NewPathIterator(vfs, c.path)
			i := 0

			for ; pi.Next(); i++ {
				if i >= len(c.parts) {
					continue
				}

				if pi.Part() != c.parts[i] {
					t.Errorf("%s : want part %d to be %s, got %s", c.path, i, c.parts[i], pi.Part())
				}

				wantLeft := pi.Path()[:pi.Start()]
				if pi.Left() != wantLeft {
					t.Errorf("%s : want left %d to be %s, got %s", c.path, i, wantLeft, pi.Left())
				}

				wantLeftPart := pi.Path()[:pi.End()]
				if pi.LeftPart() != wantLeftPart {
					t.Errorf("%s : want left %d to be %s, got %s", c.path, i, wantLeftPart, pi.LeftPart())
				}

				wantRight := pi.Path()[pi.End():]
				if pi.Right() != wantRight {
					t.Errorf("%s : want right %d to be %s, got %s", c.path, i, wantRight, pi.Right())
				}

				wantRightPart := pi.Path()[pi.Start():]
				if pi.RightPart() != wantRightPart {
					t.Errorf("%s : want right %d to be %s, got %s", c.path, i, wantRightPart, pi.RightPart())
				}

				wantIsLast := i == (len(c.parts) - 1)
				if pi.IsLast() != wantIsLast {
					t.Errorf("%s : want IsLast %d to be %t, got %t", c.path, i, wantIsLast, pi.IsLast())
				}
			}

			if i != len(c.parts) {
				t.Errorf("%s : want %d parts, got %d parts", pi.Path(), len(c.parts), i)
			}

			if c.osType == avfs.OsWindows {
				if pi.VolumeNameLen() == 0 || pi.VolumeName() == "" {
					t.Errorf("%s : want VolumeName != '', got ''", pi.Path())
				}
			} else {
				if pi.VolumeNameLen() != 0 || pi.VolumeName() != "" {
					t.Errorf("%s : want VolumeName == '', got %s ", pi.Path(), pi.VolumeName())
				}
			}
		}
	})

	t.Run("ReplacePart", func(t *testing.T) {
		cases := []struct {
			path    string
			part    string
			newPart string
			newPath string
			reset   bool
			osType  avfs.OSType
		}{
			{
				path: `c:\path`, part: `path`, newPart: `..\..\..`,
				newPath: `c:\`, reset: true, osType: avfs.OsWindows,
			},
			{
				path: `c:\an\absolute\path`, part: `absolute`, newPart: `c:\just\another`,
				newPath: `c:\just\another\path`, reset: true, osType: avfs.OsWindows,
			},
			{
				path: `c:\a\random\path`, part: `random`, newPart: `very\long`,
				newPath: `c:\a\very\long\path`, reset: false, osType: avfs.OsWindows,
			},
			{
				path: "/a/very/very/long/path", part: "long", newPart: "/a",
				newPath: "/a/path", reset: true, osType: avfs.OsLinux,
			},
			{
				path: "/path", part: "path", newPart: "../../..",
				newPath: "/", reset: true, osType: avfs.OsLinux,
			},
			{
				path: "/a/relative/path", part: "relative", newPart: "../../..",
				newPath: "/path", reset: true, osType: avfs.OsLinux,
			},
			{
				path: "/an/absolute/path", part: "random", newPart: "/just/another",
				newPath: "/just/another/path", reset: true, osType: avfs.OsLinux,
			},
			{
				path: "/a/relative/path", part: "relative", newPart: "very/long",
				newPath: "/a/very/long/path", reset: false, osType: avfs.OsLinux,
			},
		}

		for _, c := range cases {
			if c.osType != avfs.CurrentOSType() {
				continue
			}

			pi := avfs.NewPathIterator(vfs, c.path)
			for pi.Next() {
				if pi.Part() == c.part {
					reset := pi.ReplacePart(c.newPart)

					if pi.Path() != c.newPath {
						t.Errorf("%s : want new path to be %s, got %s", c.path, c.newPath, pi.Path())
					}

					if reset != c.reset {
						t.Errorf("%s : want Reset to be %t, got %t", c.path, c.reset, reset)
					}

					break
				}
			}
		}
	})
}
