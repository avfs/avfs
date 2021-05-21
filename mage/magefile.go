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

// avfs is the build script for AVFS.
package main

import (
	"fmt"
	"go/build"
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
	dockerCmd      = "docker"
	goFumptCmd     = "gofumpt"
	gitCmd         = "git"
	golangCiCmd    = "golangci-lint"
	golangCiGit    = "github.com/golangci/golangci-lint"
	golangCiBin    = "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
	goCmd          = "go"
	dockerImage    = "avfs-docker"
	coverDir       = "./coverage"
	coverFile      = "coverage.txt"
	dockerCoverDir = "/go/src/coverage"
	raceCount      = 5
	benchCount     = 5
)

var (
	cwd        string
	coverPath  string
	coverMount string
)

func init() {
	cwd, _ = os.Getwd()
	coverPath = coverDir + "/" + coverFile
	coverMount = filepath.Join(cwd, coverDir) + ":" + dockerCoverDir
}

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

// Lint runs golangci-lint (on Windows it must be run from a bash shell like git bash).
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

		err = sh.RunV("sh", script, "-b", build.Default.GOPATH+"/bin", version)
		if err != nil {
			return err
		}
	}

	return sh.RunV(golangCiCmd, "run", "-v")
}

// CoverInit resets the coverage file.
func CoverInit() error {
	err := os.MkdirAll(coverDir, 0o777)
	if err != nil {
		return err
	}

	err = os.WriteFile(coverPath, nil, 0o666)
	if err != nil {
		return err
	}

	return os.Chmod(coverPath, 0o666)
}

// Cover opens a web browser with the latest coverage file.
func Cover() error {
	if isCI() {
		return nil
	}

	return sh.RunV(goCmd, "tool", "cover", "-html="+coverPath)
}

// Test runs tests with coverage.
func Test() error {
	mg.Deps(CoverInit)

	err := sh.RunV(goCmd, "test",
		"-run=.",
		"-race", "-v",
		"-covermode=atomic",
		"-coverprofile="+coverPath,
		"./...")
	if err != nil {
		return err
	}

	return Cover()
}

// Race runs data race tests.
func Race() error {
	return sh.RunV(goCmd, "test",
		"-tags=datarace",
		"-run=TestRace",
		"-race", "-v",
		"-count="+strconv.Itoa(raceCount),
		"./...")
}

// Bench runs benchmarks.
func Bench() error {
	return sh.RunV(goCmd, "test",
		"-run=^a",
		"-bench=.",
		"-benchmem",
		"-count="+strconv.Itoa(benchCount),
		"./...")
}

// DockerBuild builds docker image for AVFS.
func DockerBuild() error {
	if !isExecutable(dockerCmd) {
		fmt.Errorf("can't find %s in the current path", dockerCmd)
	}

	return sh.RunV(dockerCmd, "build", ".", "-t", dockerImage)
}

// DockerConsole opens a shell as root in the docker image for AVFS.
func DockerConsole() error {
	mg.Deps(DockerBuild)

	return sh.RunV(dockerCmd, "run",
		"-ti",
		"-v", coverMount,
		dockerImage,
		"/bin/sh")
}

// DockerTest runs tests in the docker image for AVFS.
func DockerTest() error {
	mg.Deps(DockerBuild, CoverInit)

	err := sh.RunV(dockerCmd, "run",
		"-ti",
		"-v", coverMount,
		dockerImage)

	if err != nil {
		return err
	}

	return Cover()
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

	out, err := sh.Output(gitCmd, "ls-remote",
		"--tags",
		"--refs",
		"--sort=v:refname",
		repo)

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
