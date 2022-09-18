// Code generated by "stringer -type OSType -linecomment -output avfs_ostype.go"; DO NOT EDIT.

package avfs

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[OsUnknown-0]
	_ = x[OsLinux-1]
	_ = x[OsWindows-2]
	_ = x[OsDarwin-3]
}

const _OSType_name = "UnknownLinuxWindowsDarwin"

var _OSType_index = [...]uint8{0, 7, 12, 19, 25}

func (i OSType) String() string {
	if i >= OSType(len(_OSType_index)-1) {
		return "OSType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _OSType_name[_OSType_index[i]:_OSType_index[i+1]]
}
