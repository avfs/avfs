//go:build !go1.23

package avfs

import _ "unsafe" // for go:linkname only.

//go:linkname volumeNameLen path/filepath.volumeNameLen
func volumeNameLen(path string) int
