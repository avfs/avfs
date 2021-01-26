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

// +build linux

package osidm

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/avfs/avfs"
)

const (
	groupFile = "/etc/group"
	userFile  = "/etc/passwd"
)

var colon = []byte{':'} //nolint:gochecknoglobals // Used in matchGroupIndexValue and matchUserIndexValue.

// lineFunc returns a value, an error, or (nil, nil) to skip the row.
type lineFunc func(line []byte) (v interface{}, err error)

// readColonFile parses r as an /etc/group or /etc/passwd style file, running
// fn for each row. readColonFile returns a value, an error, or (nil, nil) if
// the end of the file is reached without a match.
func readColonFile(r io.Reader, fn lineFunc) (v interface{}, err error) {
	bs := bufio.NewScanner(r)
	for bs.Scan() {
		line := bs.Bytes()
		// There's no spec for /etc/passwd or /etc/group, but we try to follow
		// the same rules as the glibc parser, which allows comments and blank
		// space at the beginning of a line.
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		v, err = fn(line)
		if v != nil || err != nil {
			return
		}
	}

	return nil, bs.Err()
}

func matchGroupIndexValue(value string, idx int) lineFunc {
	var leadColon string
	if idx > 0 {
		leadColon = ":"
	}

	substr := []byte(leadColon + value + ":")

	return func(line []byte) (v interface{}, err error) {
		if !bytes.Contains(line, substr) || bytes.Count(line, colon) < 3 {
			return
		}
		// wheel:*:0:root
		parts := strings.SplitN(string(line), ":", 4)
		if len(parts) < 4 || parts[0] == "" || parts[idx] != value ||
			// If the file contains +foo and you search for "foo", glibc
			// returns an "invalid argument" error. Similarly, if you search
			// for a gid for a row where the group name starts with "+" or "-",
			// glibc fails to find the record.
			parts[0][0] == '+' || parts[0][0] == '-' {
			return
		}

		gid, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, nil
		}

		return &Group{name: parts[0], gid: gid}, nil
	}
}

func findGroupId(gid int, r io.Reader) (*Group, error) {
	sGid := strconv.Itoa(gid)
	if v, err := readColonFile(r, matchGroupIndexValue(sGid, 2)); err != nil {
		return nil, err
	} else if v != nil {
		return v.(*Group), nil
	}

	return nil, avfs.UnknownGroupIdError(gid)
}

func findGroupName(name string, r io.Reader) (*Group, error) {
	if v, err := readColonFile(r, matchGroupIndexValue(name, 0)); err != nil {
		return nil, err
	} else if v != nil {
		return v.(*Group), nil
	}

	return nil, avfs.UnknownGroupError(name)
}

// returns a *User for a row if that row's has the given value at the
// given index.
func matchUserIndexValue(value string, idx int) lineFunc {
	var leadColon string
	if idx > 0 {
		leadColon = ":"
	}

	substr := []byte(leadColon + value + ":")

	return func(line []byte) (v interface{}, err error) {
		if !bytes.Contains(line, substr) || bytes.Count(line, colon) < 6 {
			return
		}

		// kevin:x:1005:1006::/home/kevin:/usr/bin/zsh
		parts := strings.SplitN(string(line), ":", 7)
		if len(parts) < 6 || parts[idx] != value || parts[0] == "" ||
			parts[0][0] == '+' || parts[0][0] == '-' {
			return
		}

		uid, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, nil
		}

		gid, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, nil
		}

		u := &User{
			name: parts[0],
			uid:  uid,
			gid:  gid,
		}

		// The pw_gecos field isn't quite standardized. Some docs
		// say: "It is expected to be a comma separated list of
		// personal data where the first item is the full name of the
		// user."
		if i := strings.Index(u.name, ","); i >= 0 {
			u.name = u.name[:i]
		}

		return u, nil
	}
}

func findUserId(uid int, r io.Reader) (*User, error) {
	sUid := strconv.Itoa(uid)
	if v, err := readColonFile(r, matchUserIndexValue(sUid, 2)); err != nil {
		return nil, err
	} else if v != nil {
		return v.(*User), nil
	}

	return nil, avfs.UnknownUserIdError(uid)
}

func findUsername(name string, r io.Reader) (*User, error) {
	if v, err := readColonFile(r, matchUserIndexValue(name, 0)); err != nil {
		return nil, err
	} else if v != nil {
		return v.(*User), nil
	}

	return nil, avfs.UnknownUserError(name)
}

func lookupGroup(groupname string) (*Group, error) {
	f, err := os.Open(groupFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return findGroupName(groupname, f)
}

func lookupGroupId(gid int) (*Group, error) {
	f, err := os.Open(groupFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return findGroupId(gid, f)
}

func lookupUser(username string) (*User, error) {
	f, err := os.Open(userFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return findUsername(username, f)
}

func lookupUserId(uid int) (*User, error) {
	f, err := os.Open(userFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return findUserId(uid, f)
}

func currentUser() *User {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	uid := syscall.Geteuid()

	user, err := lookupUserId(uid)
	if err != nil {
		return nil
	}

	return user
}
