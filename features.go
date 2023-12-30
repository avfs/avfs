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

package avfs

// Features defines the set of features available on a file system.
type Features uint64

//go:generate stringer -type Features -trimprefix Feat -bitmask -output features_string.go

const (
	// FeatHardlink indicates that the file system supports hard links (link(), readlink() functions).
	FeatHardlink Features = 1 << iota

	// FeatIdentityMgr indicates that the file system features and identity manager and supports multiple users.
	FeatIdentityMgr

	// FeatSetOSType is set if the OS of the emulated file system can be changed (see MemFS).
	FeatSetOSType

	// FeatReadOnly is set for read only file systems (see RoFs).
	FeatReadOnly

	// FeatReadOnlyIdm is set when identity manager is read only (see OsIdm).
	FeatReadOnlyIdm

	// FeatRealFS indicates that the file system is a real one, not emulated (see OsFS).
	FeatRealFS

	// FeatSubFS allow to create a new file system from the subtree rooted at an arbitrary directory.
	FeatSubFS

	// FeatSymlink indicates that the file system supports symbolic links (symlink(), evalSymlink() functions).
	FeatSymlink

	// FeatSystemDirs is set when system directories  (/home, /root and /tmp for linux).
	FeatSystemDirs
)

// Featurer is the interface that wraps the Features and HasFeature methods.
type Featurer interface {
	// Features returns the set of features provided by the file system or identity manager.
	Features() Features

	// HasFeature returns true if the file system or identity manager provides a given feature.
	HasFeature(feature Features) bool
}

// FeaturesFn provides features functions to a file system or an identity manager.
type FeaturesFn struct {
	features Features // features defines the list of features available.
}

// Features returns the set of features provided by the file system or identity manager.
func (ftf *FeaturesFn) Features() Features {
	return ftf.features
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (ftf *FeaturesFn) HasFeature(feature Features) bool {
	return ftf.features&feature == feature
}

// SetFeatures sets the features of the file system or identity manager.
func (ftf *FeaturesFn) SetFeatures(feature Features) error {
	ftf.features = feature

	return nil
}

// BuildFeatures returns the features available depending on build tags.
func BuildFeatures() Features {
	return buildFeatSetOSType
}
