// Code generated by "stringer -type LinuxError -linecomment -output errors_forlinux.go"; DO NOT EDIT.

package avfs

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ErrBadFileDesc-9]
	_ = x[ErrDirNotEmpty-39]
	_ = x[ErrFileExists-17]
	_ = x[ErrInvalidArgument-22]
	_ = x[ErrIsADirectory-21]
	_ = x[ErrNoSuchFileOrDir-2]
	_ = x[ErrNotADirectory-20]
	_ = x[ErrOpNotPermitted-1]
	_ = x[ErrPermDenied-13]
	_ = x[ErrTooManySymlinks-40]
}

const (
	_LinuxError_name_0 = "operation not permittedno such file or directory"
	_LinuxError_name_1 = "bad file descriptor"
	_LinuxError_name_2 = "permission denied"
	_LinuxError_name_3 = "file exists"
	_LinuxError_name_4 = "not a directoryis a directoryinvalid argument"
	_LinuxError_name_5 = "directory not emptytoo many levels of symbolic links"
)

var (
	_LinuxError_index_0 = [...]uint8{0, 23, 48}
	_LinuxError_index_4 = [...]uint8{0, 15, 29, 45}
	_LinuxError_index_5 = [...]uint8{0, 19, 52}
)

func (i LinuxError) String() string {
	switch {
	case 1 <= i && i <= 2:
		i -= 1
		return _LinuxError_name_0[_LinuxError_index_0[i]:_LinuxError_index_0[i+1]]
	case i == 9:
		return _LinuxError_name_1
	case i == 13:
		return _LinuxError_name_2
	case i == 17:
		return _LinuxError_name_3
	case 20 <= i && i <= 22:
		i -= 20
		return _LinuxError_name_4[_LinuxError_index_4[i]:_LinuxError_index_4[i+1]]
	case 39 <= i && i <= 40:
		i -= 39
		return _LinuxError_name_5[_LinuxError_index_5[i]:_LinuxError_index_5[i+1]]
	default:
		return "LinuxError(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}