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

package avfs

import (
	"io/fs"
	"runtime"
	"sync"
)

// Config is the global configuration.
type Config struct {
	bufPool *sync.Pool // bufPool is the buffer pool used to copy files.
	bufSize int        // bufSize is the size of each buffer used to copy files.
	umask   UMaskType  // UMask is the file mode creation mask.
	utils   Utils      // Utils regroups common functions used by emulated file systems.
}

var Cfg = NewConfig() //nolint:gochecknoglobals // Cfg is the global configuration.

func NewConfig() *Config {
	var osType OSType

	switch runtime.GOOS {
	case "linux":
		osType = OsLinux
	case "darwin":
		osType = OsDarwin
	case "windows":
		osType = OsWindows
	default:
		osType = OsUnknown
	}

	cfg := Config{
		bufSize: 32 * 1024,
		utils:   NewUtils(osType),
	}

	cfg.umask.Get()
	cfg.bufPool = &sync.Pool{New: func() interface{} {
		buf := make([]byte, cfg.bufSize)

		return &buf
	}}

	return &cfg
}

// OSType returns the current Operating System type.
func (cfg *Config) OSType() OSType {
	return cfg.utils.osType
}

func (cfg *Config) Utils() Utils {
	return cfg.utils
}

func (cfg *Config) UMask() fs.FileMode {
	return cfg.umask.Get()
}

func (cfg *Config) UMaskSet(mask fs.FileMode) {
	cfg.umask.Set(mask)
}

func (cfg *Config) User() UserReader {
	return AdminUser
}
