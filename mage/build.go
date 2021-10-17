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
	"go/build"
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

	os.Setenv("CGO_ENABLED", "0")

	if isExecutable("mage") {
		log.Printf("mage binary already exists")
	} else {
		tmpDir, err := os.MkdirTemp("", "mage")
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

	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = build.Default.GOPATH
	}

	goPathDir := filepath.Join(goPath, "bin")

	err = os.MkdirAll(goPathDir, 0o777)
	if err != nil {
		log.Fatalf("MkdirAll %s : want error to be nil, got %v", goPathDir, err)
	}

	avfsBin := filepath.Join(goPathDir, "avfs")
	if runtime.GOOS == "windows" {
		avfsBin += ".exe"
	}

	err = run("mage", "-compile", avfsBin)
	if err != nil {
		log.Fatalf("mage compile : want error to be nil, got %v", err)
	}

	err = run(avfsBin, "-l")
	if err != nil {
		log.Fatalf("avfs : want error to be nil, got %v", err)
	}
}

// run executes a command cmd with arguments args.
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
