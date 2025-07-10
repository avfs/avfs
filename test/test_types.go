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

package test

import (
	"github.com/avfs/avfs"
)

const (
	defaultDir         = "defaultDir"
	defaultFile        = "defaultFile"
	defaultNonExisting = "defaultNonExisting"
)

// Suite is a test suite for virtual file systems and identity managers.
type Suite struct {
	vfsSetup    avfs.VFSBase       // vfsSetup is the file system used to set up the tests (generally with read/write access).
	vfsTest     avfs.VFSBase       // vfsTest is the file system used to run the tests.
	idm         avfs.IdentityMgr   // idm is the identity manager to be tested.
	initUser    avfs.UserReader    // initUser is the initial user running the test suite.
	rootDir     string             // rootDir is the root directory for tests and benchmarks.
	testDataDir string             // testDataDir is the testdata directory of the test suite.
	users       []avfs.UserReader  // users contains the test users created with the identity manager.
	groups      []avfs.GroupReader // groups contains the test groups created with the identity manager.
	maxRace     int                // maxRace is the maximum number of concurrent goroutines used in race tests.
	canTestPerm bool               // canTestPerm indicates if permissions can be tested.
}

// PermTests regroups all tests for a specific function.
type PermTests struct {
	ts            *Suite                // ts is a test suite for virtual file systems.
	errors        map[string]*permError // errors store the errors for each "User/Permission" combination.
	errFileName   string                // errFileName is the json file storing the test results from Ots.
	permDir       string                // permDir is the root directory of the test environment.
	options       PermOptions           // PermOptions are options for running the tests.
	errFileExists bool                  // errFileExists indicates if errFileName exits.
}

// PermOptions are options for running the tests.
type PermOptions struct {
	IgnoreOp    bool // IgnoreOp ignores the Op field comparison of fs.PathError or os.LinkError structs.
	IgnorePath  bool // IgnorePath ignores the Path, Old or New field comparison of fs.PathError or os.LinkError errors.
	CreateFiles bool // CreateFiles creates files instead of directories.
}

// errType defines the error type.
type errType string

const (
	LinkError   errType = "LinkError"
	PathError   errType = "PathError"
	StringError errType = "StringError"
)

// permError is the error returned by each test, to be stored as json.
type permError struct {
	ErrType errType `json:"errType,omitempty"`
	ErrOp   string  `json:"errOp,omitempty"`
	ErrPath string  `json:"errPath,omitempty"`
	ErrOld  string  `json:"errOld,omitempty"`
	ErrNew  string  `json:"errNew,omitempty"`
	ErrErr  string  `json:"errErr,omitempty"`
}
