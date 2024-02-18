// Code generated by "stringer -type Features -trimprefix Feat -bitmask -output features_string.go"; DO NOT EDIT.

package avfs

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[FeatHardlink-1]
	_ = x[FeatIdentityMgr-2]
	_ = x[FeatSetOSType-4]
	_ = x[FeatReadOnly-8]
	_ = x[FeatReadOnlyIdm-16]
	_ = x[FeatRealFS-32]
	_ = x[FeatSubFS-64]
	_ = x[FeatSymlink-128]
}

const _Features_name = "HardlinkIdentityMgrSetOSTypeReadOnlyReadOnlyIdmRealFSSubFSSymlink"

var _Features_map = map[Features]string{
	1:   _Features_name[0:8],
	2:   _Features_name[8:19],
	4:   _Features_name[19:28],
	8:   _Features_name[28:36],
	16:  _Features_name[36:47],
	32:  _Features_name[47:53],
	64:  _Features_name[53:58],
	128: _Features_name[58:65],
}

func (i Features) String() string {
	if i <= 0 {
		return "Features()"
	}
	sb := make([]byte, 0, len(_Features_name)/2)
	sb = append(sb, []byte("Features(")...)
	for mask := Features(1); mask > 0 && mask <= i; mask <<= 1 {
		val := i & mask
		if val == 0 {
			continue
		}
		str, ok := _Features_map[val]
		if !ok {
			str = "0x" + strconv.FormatUint(uint64(val), 16)
		}
		sb = append(sb, []byte(str)...)
		sb = append(sb, '|')
	}
	sb[len(sb)-1] = ')'
	return string(sb)
}
