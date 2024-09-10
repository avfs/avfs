//go:build go1.23

package avfs

import _ "unsafe" // for go:linkname only.

//go:linkname volumeNameLen internal/filepathlite.volumeNameLen
func volumeNameLen(path string) int
