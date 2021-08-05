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

//go:build !datarace
// +build !datarace

package avfs_test

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
	"github.com/avfs/avfs/vfs/memfs"
)

func TestAvfsErrors(t *testing.T) {
	groupName := "groupName"
	aegErr := avfs.AlreadyExistsGroupError(groupName)

	wantErrStr := "group: group " + groupName + " already exists"
	if aegErr.Error() != wantErrStr {
		t.Errorf("AlreadyExistsGroupError : want error to be %s, got %s", wantErrStr, aegErr.Error())
	}

	userName := "userName"
	wantErrStr = "user: user " + userName + " already exists"

	aeuErr := avfs.AlreadyExistsUserError(userName)
	if aeuErr.Error() != wantErrStr {
		t.Errorf("AlreadyExistsUserError : want error to be %s, got %s", wantErrStr, aeuErr.Error())
	}

	errStr := "whatever error"
	uErr := avfs.UnknownError(errStr)

	wantErrStr = "unknown error " + reflect.TypeOf(uErr).String() + " : '" + errStr + "'"
	if uErr.Error() != wantErrStr {
		t.Errorf("UnknownError : want error to be %s, got %s", wantErrStr, uErr.Error())
	}

	wantErrStr = "group: unknown group " + groupName

	ugErr := avfs.UnknownGroupError(groupName)
	if ugErr.Error() != wantErrStr {
		t.Errorf("UnknownGroupError : want error to be %s, got %s", wantErrStr, ugErr.Error())
	}

	gid := -1
	wantErrStr = "group: unknown groupid " + strconv.Itoa(gid)

	ugiErr := avfs.UnknownGroupIdError(gid)
	if ugiErr.Error() != wantErrStr {
		t.Errorf("UnknownGroupIdError : want error to be %s, got %s", wantErrStr, ugiErr.Error())
	}

	wantErrStr = "user: unknown user " + userName

	uuErr := avfs.UnknownUserError(userName)
	if uuErr.Error() != wantErrStr {
		t.Errorf("UnknownUserError : want error to be %s, got %s", wantErrStr, uuErr.Error())
	}

	uid := -1
	wantErrStr = "user: unknown userid " + strconv.Itoa(uid)

	uuiErr := avfs.UnknownUserIdError(uid)
	if uuiErr.Error() != wantErrStr {
		t.Errorf("UnknownUserIdError : want error to be %s, got %s", wantErrStr, uuiErr.Error())
	}
}

func TestAvfsMemFS(t *testing.T) {
	vfs, err := memfs.New(memfs.WithMainDirs(), memfs.WithIdm(memidm.New()))
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestVFSUtils(t)
}

func TestAvfsBaseFS(t *testing.T) {
	vfs, err := avfs.NewBaseFS()
	if err != nil {
		t.Fatalf("New : want error to be nil, got %v", err)
	}

	sfs := test.NewSuiteFS(t, vfs)
	sfs.TestVFSUtils(t)
}
