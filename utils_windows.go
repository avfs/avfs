//
//  Copyright 2021 The AVFS authors
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

//go:build windows
// +build windows

package avfs

import (
	"syscall"
)

// ShortPathName Retrieves the short path form of the specified path (Windows only).
func ShortPathName(path string) string {
	p, err := syscall.UTF16FromString(path)
	if err != nil {
		return path
	}

	b := p // GetShortPathName says we can reuse buffer
	n := uint32(len(b))

	for {
		n, err = syscall.GetShortPathName(&p[0], &b[0], n)
		if err != nil {
			return path
		}

		if n <= uint32(len(b)) {
			return syscall.UTF16ToString(b[:n])
		}

		b = make([]uint16, n)
	}
}
