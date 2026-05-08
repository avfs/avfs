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
	goCmd      = "go"
	mageCmd    = "mage"
	magePkgUrl = "github.com/magefile/mage@v1.17.2"
)

func main() {
	_, file, _, _ := runtime.Caller(0)
	mageDir := filepath.Dir(file)

	os.Setenv("CGO_ENABLED", "0")

	err := buildMage()
	if err != nil {
		log.Fatalf("buildMage : want error to be nil, got %v", err)
	}

	err = os.Chdir(mageDir)
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

	err = run(mageCmd, "-ldflags", "-w -s", "-compile", avfsBin)
	if err != nil {
		log.Fatalf("mage compile : want error to be nil, got %v", err)
	}

	err = run(avfsBin, "-l")
	if err != nil {
		log.Fatalf("avfs : want error to be nil, got %v", err)
	}
}

// buildMage builds the mage binary if it does not exist.
func buildMage() error {
	if isExecutable(mageCmd) {
		log.Printf("mage binary already exists")

		return nil
	}

	err := run(goCmd, "install", magePkgUrl)
	if err != nil {
		log.Fatalf("go install %s : want error to be nil, got %v", magePkgUrl, err)
	}

	return err
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
