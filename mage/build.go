//
//  Copyright 2020 The AVFS authors
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

//go:build ignore
// +build ignore

// Download, build and install Mage and Avfs binaries in $GOPATH/bin.
// Use "go run build.go" to install Mage.
package main

import (
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	gitCmd     = "git"
	goCmd      = "go"
	mageGitUrl = "https://github.com/magefile/mage"
)

func main() {
	_, file, _, _ := runtime.Caller(0)
	mageDir := filepath.Dir(file)

	if isExecutable("mage") {
		log.Printf("mage binary already exists")
	} else {
		tmpDir, err := ioutil.TempDir("", "mage")
		if err != nil {
			log.Fatalf("MkdirTemp : want error to be nil, got %v", err)
		}

		defer os.RemoveAll(tmpDir)

		err = os.Chdir(tmpDir)
		if err != nil {
			log.Fatalf("Chdir : want error to be nil, got %v", err)
		}

		err = run(gitCmd, "clone", "--depth=1", mageGitUrl)
		if err != nil {
			log.Fatalf("Git : want error to be nil, got %v", err)
		}

		err = os.Chdir("mage")
		if err != nil {
			log.Fatalf("Chdir : want error to be nil, got %v", err)
		}

		err = run(goCmd, "run", "bootstrap.go")
		if err != nil {
			log.Fatalf("Bootstap : want error to be nil, got %v", err)
		}
	}

	err := os.Chdir(mageDir)
	if err != nil {
		log.Fatalf("Chdir : want error to be nil, got %v", err)
	}

	avfsExe := "avfs"
	if runtime.GOOS == "windows" {
		avfsExe += ".exe"
	}

	avfsDir := filepath.Join(filepath.Dir(mageDir), "bin")

	err = os.MkdirAll(avfsDir, 0o777)
	if err != nil {
		log.Fatalf("MkdirAll %s : want error to be nil, got %v", avfsDir, err)
	}

	avfsBin := filepath.Join(avfsDir, avfsExe)

	err = run("mage", "-compile", avfsBin)
	if err != nil {
		log.Fatalf("mage compile : want error to be nil, got %v", err)
	}

	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = build.Default.GOPATH
	}

	goPathDir := filepath.Join(goPath, "bin")
	goPathBin := filepath.Join(goPathDir, avfsExe)

	err = os.MkdirAll(goPathDir, 0o777)
	if err != nil {
		log.Fatalf("MkdirAll %s : want error to be nil, got %v", goPathDir, err)
	}

	err = copy(goPathBin, avfsBin)
	if err != nil {
		log.Fatalf("copy %s %s : want error to be nil, got %v", goPathBin, avfsBin, err)
	}

	err = run(avfsExe, "-l")
	if err != nil {
		log.Fatalf("avfs : want error to be nil, got %v", err)
	}
}

// run runs a command cmd with arguments args.
func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout

	return c.Run()
}

// isExecutable checks if name is an executable in the current path.
func isExecutable(name string) bool {
	_, err := exec.LookPath(name)

	return err == nil
}

// copy robustly copies the source file to the destination, overwriting the destination if necessary.
func copy(dst, src string) error {
	from, err := os.Open(src)
	if err != nil {
		return fmt.Errorf(`can't copy %s: %v`, src, err)
	}

	defer from.Close()

	finfo, err := from.Stat()
	if err != nil {
		return fmt.Errorf(`can't stat %s: %v`, src, err)
	}

	to, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, finfo.Mode())
	if err != nil {
		return fmt.Errorf(`can't copy to %s: %v`, dst, err)
	}

	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return fmt.Errorf(`error copying %s to %s: %v`, src, dst, err)
	}

	return nil
}
