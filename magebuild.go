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

// +build magebuild

// Download, build and install Mage in $GOPATH/bin.
// Use "go run magebuild.go" to install Mage.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func main() {
	const mageGitUrl = "https://github.com/magefile/mage"

	if isExecutable("mage") {
		os.Exit(0)
	}

	rootDir, err := ioutil.TempDir("", "mage")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TempDir : want error to be nil, got %v", err)
		os.Exit(1)
	}

	defer os.RemoveAll(rootDir)

	err = os.Chdir(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Chdir : want error to be nil, got %v", err)
		os.Exit(2)
	}

	err = run("git", "clone", mageGitUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Git : want error to be nil, got %v", err)
		os.Exit(3)
	}

	err = os.Chdir("mage")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Chdir : want error to be nil, got %v", err)
		os.Exit(4)
	}

	err = run("go", "run", "bootstrap.go")
	if err != nil {
		fmt.Fprintf(os.Stderr, "go run : want error to be nil, got %v", err)
		os.Exit(5)
	}

	err = run("mage", "--version")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Mage : want error to be nil, got %v", err)
		os.Exit(6)
	}

	os.Exit(0)
}

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
