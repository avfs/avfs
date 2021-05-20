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

// +build ignore

// Download, build and install Mage and Avfs binaries in $GOPATH/bin.
// Use "go run build.go" to install Mage.
package main

import (
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const mageGitUrl = "https://github.com/magefile/mage"

func main() {
	if isExecutable("mage") {
		log.Printf("mage binary already exists")
	}

	appDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Getwd : want error to be nil, got %v", err)
	}

	if !isExecutable("mage") {
		tmpDir, err := ioutil.TempDir("", "mage")
		if err != nil {
			log.Fatalf("TempDir : want error to be nil, got %v", err)
		}

		defer os.RemoveAll(tmpDir)

		err = os.Chdir(tmpDir)
		if err != nil {
			log.Fatalf("Chdir : want error to be nil, got %v", err)
		}

		err = run("git", "clone", "--depth=1", mageGitUrl)
		if err != nil {
			log.Fatalf("Git : want error to be nil, got %v", err)
		}

		err = os.Chdir("mage")
		if err != nil {
			log.Fatalf("Chdir : want error to be nil, got %v", err)
		}

		err = run("go", "run", "bootstrap.go")
		if err != nil {
			log.Fatalf("Bootstap : want error to be nil, got %v", err)
		}
	}

	mageDir := filepath.Join(appDir, "mage")

	err = os.Chdir(mageDir)
	if err != nil {
		log.Fatalf("Chdir : want error to be nil, got %v", err)
	}

	binDir := filepath.Join(build.Default.GOPATH, "bin")

	err = os.MkdirAll(binDir, 0o755)
	if err != nil {
		log.Fatalf("MkdirAll : want error to be nil, got %v", err)
	}

	avfsBin := "avfs"
	if runtime.GOOS == "windows" {
		avfsBin += ".exe"
	}

	avfsPath := filepath.Join(binDir, avfsBin)

	err = run("mage", "-compile", avfsPath)
	if err != nil {
		log.Fatalf("mage compile : want error to be nil, got %v", err)
	}

	err = run("avfs", "-l")
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
