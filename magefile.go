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

// +build mage

// This is the build script for AVFS.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	dockerCmd    = "docker"
	goFumptCmd   = "gofumpt"
	gitCmd       = "git"
	golangCiCmd  = "golangci-lint"
	golangCiGit  = "github.com/golangci/golangci-lint"
	golangCiBin  = "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
	goCmd        = "go"
	mageCmd      = "mage"
	dockerImage  = "avfs-docker"
	coverageFile = "coverage.txt"
	raceCount    = 5
	benchCount   = 5
)

// Env returns the go environment variables.
func Env() {
	sh.RunV(goCmd, "version")
	sh.RunV(goCmd, "env")
}

// Build builds the project.
func Build() error {
	return sh.RunV(goCmd, "build", "-v", "./...")
}

// Fmt runs gofumpt on the project.
func Fmt() error {
	if !isExecutable(goFumptCmd) {
		err := sh.RunV(goCmd, "get", "mvdan.cc/gofumpt")
		if err != nil {
			return err
		}
	}

	return sh.RunV(goFumptCmd, "-l", "-s", "-w", "-extra", ".")
}

// Lint runs golangci-lint (on Windows it must be run from bash shell like git bash).
func Lint() error {
	if !isExecutable(golangCiCmd) {
		version, err := gitLastVersion(golangCiGit)
		if err != nil {
			return err
		}

		fmt.Printf("version = %s\n", version)

		script := filepath.Join(os.TempDir(), golangCiCmd+".sh")

		err = downloadFile(script, golangCiBin)
		if err != nil {
			return err
		}

		defer os.Remove(script)

		gopath := os.Getenv("GOPATH")
		err = sh.RunV("sh", script, "-b", gopath+"/bin", version)
		if err != nil {
			return err
		}
	}

	return sh.RunV(golangCiCmd, "run", "-v")
}

// Cover opens a web browser with the latest coverage file.
func Cover() error {
	if isCI() {
		return nil
	}

	return sh.RunV(goCmd, "tool", "cover", "-html="+coverageFile)
}

// Test runs tests with coverage.
func Test() error {
	mg.Deps(Env)

	err := sh.Rm(coverageFile)
	if err != nil {
		return err
	}

	err = sh.RunV(goCmd, "test", "-run=.", "-race", "-v", "-covermode=atomic",
		"-coverprofile="+coverageFile, "./...")
	if err != nil {
		return err
	}

	if !isCI() {
		Cover()
	}

	return nil
}

// Race runs data race tests.
func Race() error {
	return sh.RunV(goCmd, "test", "-tags=datarace", "-run=TestRace", "-race", "-v",
		"-count="+strconv.Itoa(raceCount), "./...")
}

// Bench runs benchmarks.
func Bench() error {
	return sh.RunV(goCmd, "test", "-run=^a", "-bench=.", "-benchmem",
		"-count="+strconv.Itoa(benchCount), "./...")
}

// DockerMage builds a static mage version of these scripts to be used in docker.
func DockerMage() error {
	return sh.RunV(mageCmd, "-compile", "./bin/dockermage", "-goos=linux")
}

// DockerBuild builds docker image for AVFS.
func DockerBuild() error {
	mg.Deps(DockerMage)

	if !isExecutable(dockerCmd) {
		fmt.Errorf("can't find %s in the current path", dockerCmd)
	}

	return sh.RunV(dockerCmd, "build", ".", "-t", dockerImage)
}

// DockerConsole opens a shell as root in the docker image for AVFS.
func DockerConsole() error {
	mg.Deps(DockerBuild)

	return sh.RunV(dockerCmd, "run", "--network", "host", "-ti", dockerImage, "/bin/bash")
}

// DockerTest runs tests in the docker image for AVFS.
func DockerTest() error {
	mg.Deps(DockerBuild)

	err := sh.RunV(dockerCmd, "run", "--network", "host", "-ti", dockerImage)
	if err != nil {
		return err
	}

	container, err := sh.Output(dockerCmd, "ps", "-alq")
	if err != nil {
		return err
	}

	err = sh.RunV(dockerCmd, "cp", container+":/go/src/"+coverageFile, coverageFile)
	if err != nil {
		return err
	}

	Cover()

	return nil
}

// DockerPrune removes unused data from Docker.
func DockerPrune() error {
	return sh.RunV(dockerCmd, "system", "prune", "-f")
}

// isExecutable checks if name is an executable in the current path.
func isExecutable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// isCI tests if we run in a CI environment.
func isCI() bool {
	return os.Getenv("CI") != ""
}

// gitLastVersion return the latest tagged version of a remote git repository.
func gitLastVersion(repo string) (string, error) {
	const semverRegexp = "v\\d+\\.\\d+\\.\\d+$"

	if !strings.HasPrefix(repo, "https://") {
		repo = "https://" + repo
	}

	out, err := sh.Output(gitCmd, "ls-remote", "--tags", "--refs", "--sort=v:refname", repo)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(semverRegexp)
	version := re.FindString(out)
	if version == "" {
		return "", fmt.Errorf("version : incorrect format :\n%s", out)
	}

	return version, nil
}

// downloadFile downloads a url to a local file.
func downloadFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.Copy(f, resp.Body)

	return err
}
