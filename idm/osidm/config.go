package osidm

import "github.com/avfs/avfs"

// New creates a new OsIdm identity manager.
func New() *OsIdm {
	return &OsIdm{}
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *OsIdm) Type() string {
	return "OsIdm"
}

// Features returns the set of features provided by the file system or identity manager.
func (idm *OsIdm) Features() avfs.Feature {
	return avfs.FeatIdentityMgr
}

// HasFeatures returns true if the file system or identity manager provides all the given features.
func (idm *OsIdm) HasFeatures(feature avfs.Feature) bool {
	return avfs.FeatIdentityMgr&feature == feature
}
