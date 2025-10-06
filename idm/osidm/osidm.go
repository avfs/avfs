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

// Package osidm implements an identity manager using os functions.
//
// For testing only, should not be used in a production environment.
package osidm

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"

	"github.com/avfs/avfs"
)

// AdminGroup returns the administrator (root) group.
func (idm *OsIdm) AdminGroup() avfs.GroupReader {
	return idm.adminGroup
}

// AdminUser returns the administrator (root) user.
func (idm *OsIdm) AdminUser() avfs.UserReader {
	return idm.adminUser
}

// OsGroup

// Gid returns the group ID.
func (g *OsGroup) Gid() int {
	return g.gid
}

// Name returns the group name.
func (g *OsGroup) Name() string {
	return g.name
}

// OsUser

// Gid returns the primary group ID of the user.
func (u *OsUser) Gid() int {
	return u.gid
}

// IsAdmin returns true if the user has administrator (root) privileges.
func (u *OsUser) IsAdmin() bool {
	return u.uid == 0 || u.gid == 0
}

// Name returns the user name.
func (u *OsUser) Name() string {
	return u.name
}

// Uid returns the user ID.
func (u *OsUser) Uid() int {
	return u.uid
}

// run executes a command cmd with arguments args,
// returns the text from os.Stderr as error.
func run(cmd string, args ...string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	c := exec.Command(cmd, args...)

	var stderr bytes.Buffer

	c.Stderr = &stderr
	err := c.Run()

	return err
}

// output executes a command cmd with arguments args,
// returns the text from os.Stdout and the text from os.Stderr as error.
func output(cmd string, args ...string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	c := exec.Command(cmd, args...)

	var stderr bytes.Buffer

	c.Stderr = &stderr
	buf, err := c.Output()

	return strings.TrimSuffix(string(buf), "\n"), err
}
