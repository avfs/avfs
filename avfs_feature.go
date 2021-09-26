// Code generated by "stringer -type Feature -trimprefix Feat -bitmask -output avfs_feature.go"; DO NOT EDIT.

package avfs

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[FeatBasicFs-1]
	_ = x[FeatChroot-2]
	_ = x[FeatChownUser-4]
	_ = x[FeatMainDirs-8]
	_ = x[FeatHardlink-16]
	_ = x[FeatIdentityMgr-32]
	_ = x[FeatReadOnly-64]
	_ = x[FeatReadOnlyIdm-128]
	_ = x[FeatRealFS-256]
	_ = x[FeatSymlink-512]
}

const _Feature_name = "BasicFsChrootChownUserMainDirsHardlinkIdentityMgrReadOnlyReadOnlyIdmRealFSSymlink"

var _Feature_map = map[Feature]string{
	1:   _Feature_name[0:7],
	2:   _Feature_name[7:13],
	4:   _Feature_name[13:22],
	8:   _Feature_name[22:30],
	16:  _Feature_name[30:38],
	32:  _Feature_name[38:49],
	64:  _Feature_name[49:57],
	128: _Feature_name[57:68],
	256: _Feature_name[68:74],
	512: _Feature_name[74:81],
}

func (i Feature) String() string {
	if i <= 0 {
		return "Feature()"
	}
	sb := make([]byte, 0, len(_Feature_name)/2)
	sb = append(sb, []byte("Feature(")...)
	for mask := Feature(1); mask > 0 && mask <= i; mask <<= 1 {
		val := i & mask
		if val == 0 {
			continue
		}
		str, ok := _Feature_map[val]
		if !ok {
			str = "0x" + strconv.FormatUint(uint64(val), 16)
		}
		sb = append(sb, []byte(str)...)
		sb = append(sb, '|')
	}
	sb[len(sb)-1] = ')'
	return string(sb)
}
