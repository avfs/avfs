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

package avfs

// NewBaseIdm create a new identity manager.
func NewBaseIdm() *BaseIdm {
	return &BaseIdm{}
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *BaseIdm) Type() string {
	return "BaseIdm"
}

// Features returns the set of features provided by the file system or identity manager.
func (idm *BaseIdm) Features() Feature {
	return 0
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (idm *BaseIdm) HasFeature(feature Feature) bool {
	return false
}
