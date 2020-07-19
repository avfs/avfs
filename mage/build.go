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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func main() {
	err := buildMage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "BuildMage : %v\n", err)
		os.Exit(1)
	}

	err = buildAvfs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "BuildAvfs : %v\n", err)
		os.Exit(2)
	}

	err = run("avfs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "RunAvfs : %v\n", err)
		os.Exit(3)
	}
}

// buildMage builds mage binary and saves it in $GOPATH/bin.
func buildMage() error {
	const mageGitUrl = "https://github.com/magefile/mage"

	if isExecutable("mage") {
		return nil
	}

	appDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Getwd : want error to be nil, got %v", err)
	}

	rootDir, err := ioutil.TempDir("", "mage")
	if err != nil {
		return fmt.Errorf("TempDir : want error to be nil, got %v", err)
	}

	defer os.RemoveAll(rootDir)

	err = os.Chdir(rootDir)
	if err != nil {
		return fmt.Errorf("Chdir : want error to be nil, got %v", err)
	}

	err = run("git", "clone", mageGitUrl)
	if err != nil {
		return fmt.Errorf("Git : want error to be nil, got %v", err)
	}

	err = os.Chdir("mage")
	if err != nil {
		return fmt.Errorf("Chdir : want error to be nil, got %v", err)
	}

	err = run("go", "run", "bootstrap.go")
	if err != nil {
		return fmt.Errorf("Bootstap : want error to be nil, got %v", err)
	}

	err = os.Chdir(appDir)
	if err != nil {
		return fmt.Errorf("Chdir : want error to be nil, got %v", err)
	}

	return nil
}

// buildAvfs builds avfs binary as saves it in $GOPATH/bin.
func buildAvfs() error {
	err := run("mage", "-d", "mage", "-w", ".", "BinLocal")
	if err != nil {
		return fmt.Errorf("mage : want error to be nil, got %v", err)
	}

	return nil
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
