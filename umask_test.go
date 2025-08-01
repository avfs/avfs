//
//  Copyright 2023 The AVFS authors
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
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
)

// TestUMaskOS tests Umask functions for the current OS.
func TestUMaskOS(t *testing.T) {
	const (
		linuxUMask   = fs.FileMode(0o22)
		windowsUMask = fs.FileMode(0o111)
		testUMask    = fs.FileMode(0o77)
	)

	saveUMask := avfs.UMask()

	defer func() { _ = avfs.SetUMask(saveUMask) }()

	defaultUMask := linuxUMask
	if avfs.CurrentOSType() == avfs.OsWindows {
		defaultUMask = windowsUMask
	}

	wantedUMask := defaultUMask

	umask := avfs.UMask()
	if umask != wantedUMask {
		t.Errorf("UMask : want OS umask %o, got %o", wantedUMask, umask)
	}

	_ = avfs.SetUMask(testUMask)

	umask = avfs.UMask()
	if umask != testUMask {
		t.Errorf("UMask : want test umask %o, got %o", testUMask, umask)
	}

	_ = avfs.SetUMask(defaultUMask)

	umask = avfs.UMask()
	if umask != defaultUMask {
		t.Errorf("UMask : want OS umask %o, got %o", defaultUMask, umask)
	}
}
