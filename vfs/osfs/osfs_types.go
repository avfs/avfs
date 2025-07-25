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

package osfs

import (
	"github.com/avfs/avfs"
)

// OsFS represents the current file system.
type OsFS struct {
	err             *avfs.ErrorsForOS // err regroups errors depending on the OS emulated.
	avfs.IdmFn                        // IdmFn provides identity manager functions to a file system.
	avfs.FeaturesFn                   // FeaturesFn provides features functions to a file system or an identity manager.
}

// Options defines the initialization options of OsFS.
type Options struct {
	Idm avfs.IdentityMgr // Idm is the identity manager of the file system.
}
