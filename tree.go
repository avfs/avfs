//
//  Copyright 2024 The AVFS authors
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

import (
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

type (
	userCache  = map[int]string
	groupCache = map[int]string
)

type treeInfo struct {
	vfs     VFSBase
	builder strings.Builder
	users   userCache
	groups  groupCache
	nbDirs  int
	nbFiles int
}

// Tree returns a textual representation of the directory structure.
func Tree(vfs VFSBase, path string) string {
	ti := newTreeInfo(vfs)

	absPath, err := vfs.Abs(path)
	if err != nil {
		return ""
	}

	info, err := vfs.Stat(absPath)
	if err != nil {
		ti.addLineErr("", absPath, err)

		return ti.builder.String()
	}

	ti.addLine(info, "", info.Name())
	ti.tree("", absPath)

	line := fmt.Sprintf("\n%d directories, %d files", ti.nbDirs, ti.nbFiles)
	ti.builder.WriteString(line)

	return ti.builder.String()
}

func newTreeInfo(vfs VFSBase) *treeInfo {
	ti := &treeInfo{vfs: vfs, nbDirs: 1}
	ti.users = make(userCache)
	ti.groups = make(groupCache)

	return ti
}

func (ti *treeInfo) tree(prefix, path string) {
	vfs := ti.vfs

	dirs, err := vfs.ReadDir(path)
	if err != nil {
		ti.addLineErr(prefix, path, err)
	}

	for i, dir := range dirs {
		name := dir.Name()
		if name[0] == '.' {
			continue
		}

		var sep, prf string

		lastEntry := i == len(dirs)-1
		if lastEntry {
			sep = "└──"
			prf = "   "
		} else {
			sep = "├──"
			prf = "│  "
		}

		info, err := dir.Info()
		if err != nil {
			ti.addLineErr(prefix, name, err)

			continue
		}

		ti.addLine(info, prefix+sep, name)

		if dir.IsDir() {
			ti.nbDirs++

			absPath := vfs.Join(path, name)
			ti.tree(prefix+prf, absPath)
		} else {
			ti.nbFiles++
		}
	}
}

func (ti *treeInfo) addLine(info fs.FileInfo, prefix, path string) {
	vfs := ti.vfs

	perms := info.Mode().Perm().String()
	sys := vfs.ToSysStat(info)

	gid := sys.Gid()
	sGid := ti.groupName(gid)

	uid := sys.Uid()
	sUid := ti.userName(uid)

	line := fmt.Sprintf("%s[%s, %s, %s] %s\n", prefix, perms, sUid, sGid, path)
	ti.builder.WriteString(line)
}

func (ti *treeInfo) addLineErr(prefix, path string, err error) {
	line := fmt.Sprintf("%s[-, -, -] %s - %v\n", prefix, path, err)
	ti.builder.WriteString(line)
}

func (ti *treeInfo) groupName(gid int) string {
	name, ok := ti.groups[gid]
	if ok {
		return name
	}

	idm := ti.vfs.Idm()

	g, err := idm.LookupGroupId(gid)
	if err != nil {
		name = strconv.Itoa(gid)
	} else {
		name = g.Name()
	}

	ti.groups[gid] = name

	return name
}

func (ti *treeInfo) userName(uid int) string {
	name, ok := ti.users[uid]
	if ok {
		return name
	}

	idm := ti.vfs.Idm()

	u, err := idm.LookupUserId(uid)
	if err != nil {
		name = strconv.Itoa(uid)
	} else {
		name = u.Name()
	}

	ti.users[uid] = name

	return name
}
