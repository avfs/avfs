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

// +build bootstrap_mage

// Download an build Mage.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func main() {
	const mageGitUrl = "https://github.com/magefile/mage"

	rootDir, err := ioutil.TempDir("", "mage")
	if err != nil {
		fmt.Printf("TempDir : want error to be nil, got %v", err)
	}

	defer os.RemoveAll(rootDir)

	err = os.Chdir(rootDir)
	if err != nil {
		fmt.Printf("Chdir : want error to be nil, got %v", err)
	}

	err = run("git", "clone", "--depth", "1", mageGitUrl)
	if err != nil {
		fmt.Printf("Git : want error to be nil, got %v", err)
	}

	err = os.Chdir("mage")
	if err != nil {
		fmt.Printf("Chdir : want error to be nil, got %v", err)
	}

	err = run("go", "run", "bootstrap.go")
	if err != nil {
		fmt.Printf("Chdir : want error to be nil, got %v", err)
	}

	err = run("mage", "--version")
	if err != nil {
		fmt.Printf("Mage : want error to be nil, got %v", err)
	}
}

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout

	return c.Run()
}
