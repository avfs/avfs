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

//go:build mage

// avfs is the build script for AVFS.
package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/build"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	dockerGoSrc = "/go/src"
	dockerImage = "avfs-docker"
	gitCmd      = "git"
	goCmd       = "go"
	goFumptCmd  = "gofumpt"
	goFumptInst = "mvdan.cc/gofumpt@master"
	golangCiCmd = "golangci-lint"
	golangCiGit = "github.com/golangci/golangci-lint"
	golangCiTgz = "https://github.com/golangci/golangci-lint/releases/download/%s/golangci-lint-%s-%s-%s.tar.gz"
	golangCiPkg = "golangci-lint-%s-%s-%s.tar.gz"
	golangCiChk = "https://github.com/golangci/golangci-lint/releases/download/%s/golangci-lint-%s-checksums.txt"
	goxCmd      = "gox"
	goxInst     = "github.com/mitchellh/gox@master"
	sudoCmd     = "sudo"
	tarCmd      = "tar"
	raceCount   = 12
	benchCount  = 12
)

var (
	appDir            string
	cgoEnabled        bool
	coverFile         string
	dockerCmd         string
	dockerTestDataDir string
	dockerTmpDir      string
	goPathBinDir      string
	tmpDir            string
	testDataDir       string
)

func init() {
	appDir, _ = os.Getwd()
	appDir = strings.TrimSuffix(appDir, "mage")

	tmpDir = filepath.Join(appDir, "tmp")
	coverFile = filepath.Join(tmpDir, "avfs-cover.txt")
	testDataDir = filepath.Join(appDir, "test/testdata")

	dockerVolume := ""
	if runtime.GOOS == "windows" {
		dockerVolume = "c:"
	}

	dockerTmpDir = filepath.Join(dockerVolume, dockerGoSrc, "tmp")
	dockerTestDataDir = filepath.Join(dockerVolume, dockerGoSrc, "test/testdata")

	switch {
	case isExecutable("docker"):
		dockerCmd = "docker"
	case isExecutable("podman"):
		dockerCmd = "podman"
	default:
		dockerCmd = ""
	}

	cc := os.Getenv("CC")
	if cc == "" {
		cc = "gcc"
	}

	ccPath, err := exec.LookPath(cc)
	if err == nil && ccPath != "" {
		cgoEnabled = true
	}

	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = build.Default.GOPATH
	}

	goPathBinDir = filepath.Join(goPath, "bin")
}

// tmpInit creates the temporary directory.
func tmpInit() error {
	err := os.MkdirAll(tmpDir, 0o755)
	if err != nil {
		return err
	}

	return os.Chmod(tmpDir, 0o777)
}

// Env returns the go environment variables.
func Env() {
	sh.RunV(goCmd, "env")
	fmt.Printf(`
appDir=%s
tmpDir=%s
testDataDir=%s
dockerTmpDir=%s
dockerTestDataDir=%s
coverFile=%s
cgoEnabled=%t
`, appDir, tmpDir, testDataDir, dockerTmpDir, dockerTestDataDir, coverFile, cgoEnabled)
}

// Build builds the project.
func Build() error {
	return sh.RunV(goCmd, "build", "-v", "./...")
}

// Fmt runs gofumpt on the project.
func Fmt() error {
	if !isExecutable(goFumptCmd) {
		err := os.Chdir(os.TempDir())
		if err != nil {
			return err
		}

		err = sh.RunV(goCmd, "install", goFumptInst)
		if err != nil {
			return err
		}

		err = os.Chdir(appDir)
		if err != nil {
			return err
		}
	}

	return sh.RunV(goFumptCmd, "-l", "-w", "-extra", ".")
}

// Lint runs golangci-lint (on Windows it must be run from a bash shell like git bash).
func Lint() error {
	mg.Deps(tmpInit)

	golangCiBin := golangCiCmd
	if runtime.GOOS == "windows" {
		golangCiBin += ".exe"
	}

	// golangci-lint can be already installed in the current path or in $GOPATH/bin.
	golangCiPath := executablePath(golangCiBin)
	if golangCiPath == "" {
		version, err := gitLastVersion(golangCiGit)
		if err != nil {
			return err
		}

		golangCiPath = filepath.Join(goPathBinDir, golangCiBin)
		tgzUrl := fmt.Sprintf(golangCiTgz, version, version[1:], runtime.GOOS, runtime.GOARCH)
		// The archive is named after the release asset so that its checksum can be
		// found in the checksums file.
		tgzFile := filepath.Join(tmpDir, golangCiTgzName(version))
		chkUrl := fmt.Sprintf(golangCiChk, version, version[1:])
		chkFile := filepath.Join(tmpDir, "golangci-lint-checksums.txt")

		fmt.Printf("version = %s\ntgz url = %s\ntgz file = %s\nchk url = %s\nchk file = %s\nbin = %s\n",
			version, tgzUrl, tgzFile, chkUrl, chkFile, golangCiPath)

		err = downloadFile(chkFile, chkUrl)
		if err != nil {
			return err
		}

		defer os.Remove(chkFile)

		err = downloadFile(tgzFile, tgzUrl)
		if err != nil {
			return err
		}

		defer os.Remove(tgzFile)

		err = CheckFile(tgzFile, chkFile)
		if err != nil {
			return err
		}

		err = gzipExtract(tgzFile, golangCiBin, golangCiPath)
		if err != nil {
			return err
		}
	}

	return sh.RunV(golangCiPath, "run", "-v")
}

// golangCiTgzName returns the name of the golangci-lint archive for the current
// platform at the given version.
func golangCiTgzName(version string) string {
	return fmt.Sprintf(golangCiPkg, version[1:], runtime.GOOS, runtime.GOARCH)
}

// gzipExtract extracts a specific file from a gzip-compressed tar archive and saves it to a given destination.
// Parameters:
//   - gzipName: The name of the gzip-compressed tar archive to be opened.
//   - fileName: The name of the file to be extracted from the archive.
//   - destName: The destination path where the extracted file will be saved.
//
// Returns an error if the file cannot be extracted or if any I/O operation fails.
func gzipExtract(gzipName, fileName, destName string) error {
	f, err := os.Open(gzipName)
	if err != nil {
		return err
	}

	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if strings.HasSuffix(header.Name, fileName) {
			outFile, err := os.Create(destName)
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, tr)
			if err != nil {
				return err
			}

			err = outFile.Close()
			if err != nil {
				return err
			}

			return os.Chmod(destName, 0o755)
		}
	}

	return fmt.Errorf("file %s not found in archive %s", fileName, gzipName)
}

// CheckFile verifies the SHA256 checksum of a file against a golangci-lint checksums file.
func CheckFile(fileName, chkFile string) error {
	want, err := checksum(chkFile, fileName)
	if err != nil {
		return err
	}

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}

	defer f.Close()

	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return err
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("invalid checksum for %s: got %s, want %s", fileName, got, want)
	}

	return nil
}

// checksum returns the expected checksum of fileName found in a checksums file
// (in the "<sha256>  <filename>" format used by golangci-lint).
func checksum(chkFile, fileName string) (string, error) {
	content, err := os.ReadFile(chkFile)
	if err != nil {
		return "", err
	}

	base := filepath.Base(fileName)

	for line := range strings.SplitSeq(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		if filepath.Base(fields[1]) == base {
			return fields[0], nil
		}
	}

	return "", fmt.Errorf("checksum of %s not found in %s", fileName, chkFile)
}

// sudo runs a command as root if possible, as an unprivileged user otherwise.
func sudo(cmd string, args ...string) error {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return sh.RunV(cmd, args...)
	}

	err := sh.RunV(sudoCmd, "-n", "-l")
	if err != nil {
		return sh.RunV(cmd, args...)
	}

	sudoArgs := append([]string{"-n", "-E", "env", "PATH=$PATH", cmd}, args...)

	return sh.RunV(sudoCmd, sudoArgs...)
}

// testArgs returns the arguments of the go command used for tests.
func testArgs() []string {
	pkgs := goPackages("/test")
	args := []string{"test", "-v", "-covermode=atomic", "-coverprofile=" + coverFile}
	args = append(args, pkgs...)

	if cgoEnabled {
		args = append(args, "-race")
	}

	return args
}

// CoverResult opens a web browser with the latest coverage file if used interactively,
// or archive current coverage file when executed in CI mode.
func CoverResult() error {
	if isCI() {
		// Archive coverage file for code coverage upload.
		coverArch := filepath.Join(tmpDir, time.Now().Format("avfs-cover-20060102-030405.txt"))

		return os.Rename(coverFile, coverArch)
	}

	return sh.RunV(goCmd, "tool", "cover", "-html="+coverFile)
}

// Test runs tests with coverage as the current user.
func Test() error {
	mg.Deps(tmpInit)

	err := sh.RunV(goCmd, testArgs()...)
	if err != nil {
		return err
	}

	return CoverResult()
}

// TestAsRoot runs tests as root with coverage (using sudo if necessary).
func TestAsRoot() error {
	mg.Deps(tmpInit)

	err := sudo(goCmd, testArgs()...)
	if err != nil {
		return err
	}

	return CoverResult()
}

// goOSArch returns an array of [goos/goarch, goos, goarch].
func goOSArch(exclusions ...string) []string {
	buf, err := sh.Output(goCmd, "tool", "dist", "list")
	if err != nil {
		return nil
	}

	lines := strings.Split(buf, "\n")
	result := make([]string, 0, len(lines))

mainLoop:
	for _, line := range lines {
		if !strings.Contains(line, "/") {
			continue
		}

		for _, exclusion := range exclusions {
			if strings.Contains(line, exclusion) {
				continue mainLoop
			}
		}

		result = append(result, line)
	}

	return result
}

// goPackages list packages excluding those containing exclude text.
func goPackages(exclude string) []string {
	out, err := sh.Output(goCmd, "list", "./...")
	if err != nil {
		return nil
	}

	var lines []string
	var start int

	for i, r := range out {
		if r == '\n' {
			line := out[start:i]
			if !strings.Contains(line, exclude) {
				lines = append(lines, line)
			}

			start = i + 1
		}
	}

	line := out[start:]
	if !strings.Contains(line, exclude) {
		lines = append(lines, line)
	}

	return lines
}

// TestBuild builds a test executable on all architectures (except Android/*)
func TestBuild() error {
	mg.Deps(tmpInit)

	if !isExecutable(goxCmd) {
		err := sh.RunV(goCmd, "install", goxInst)
		if err != nil {
			return err
		}
	}

	srcPath := filepath.Join(appDir, "test/testbuild")
	outPath := filepath.Join(appDir, "tmp/{{.Dir}}/{{.Dir}}_{{.OS}}_{{.Arch}}")
	archArr := goOSArch("android/") // Exclude Android platforms: need additional tools to compile.

	fmt.Printf("\n archarr = %v\n", archArr)

	err := sh.RunV(goxCmd, "-cgo",
		"-osarch=\""+strings.Join(archArr, " ")+"\"",
		"-output="+outPath,
		srcPath)
	if err != nil {
		return err
	}

	return nil
}

// Race runs data race tests.
func Race() error {
	mg.Deps(tmpInit)

	if !cgoEnabled {
		return nil
	}

	err := sh.RunV(goCmd, "test", "-v",
		"-tags=avfs_race",
		"-run=TestRace",
		"-race",
		"-count="+strconv.Itoa(raceCount),
		"-covermode=atomic",
		"-coverprofile="+coverFile,
		"./...")
	if err != nil {
		return err
	}

	return CoverResult()
}

// Bench runs benchmarks.
func Bench() error {
	return sh.RunV(goCmd, "test", "-v",
		"-run=^a",
		"-bench=.",
		"-benchmem",
		"-count="+strconv.Itoa(benchCount),
		"./...")
}

// DockerBuild builds docker image for AVFS.
func DockerBuild() error {
	mg.Deps(tmpInit)

	if dockerCmd == "" {
		return fmt.Errorf("can't find docker or podman in the current path")
	}

	dockerFile := runtime.GOOS + ".Dockerfile"

	_, err := os.Stat(dockerFile)
	if err != nil {
		return fmt.Errorf("can't find Dockerfile %s", dockerFile)
	}

	avfsPath, err := exec.LookPath("avfs")
	if err != nil {
		return err
	}

	avfsBin := filepath.Base(avfsPath)
	tmpBin := "tmp/" + avfsBin

	err = sh.Copy(tmpBin, avfsPath)
	if err != nil {
		return err
	}

	err = sh.RunV(tarCmd, "-cf", "tmp/avfs.tar", "--exclude-vcs", "--exclude-vcs-ignores", ".")
	if err != nil {
		return err
	}

	return sh.RunV(dockerCmd,
		"build",
		"-t", dockerImage,
		"-f", dockerFile,
		".")
}

// DockerTerm opens a shell as root in the docker image for AVFS.
func DockerTerm() error {
	mg.Deps(DockerBuild)

	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "cmd"
	}

	return dockerTest(shell)
}

// DockerTest runs tests in the docker image and displays the coverage result.
func DockerTest() error {
	mg.Deps(DockerBuild)

	err := dockerTest()
	if err != nil {
		return err
	}

	return CoverResult()
}

// DockerPrune removes unused data from Docker.
func DockerPrune() error {
	return sh.RunV(dockerCmd, "system", "prune", "-f")
}

// dockerTest runs tests in the docker image for AVFS.
func dockerTest(args ...string) error {
	termOptions := "-it"
	if runtime.GOOS == "windows" {
		termOptions = "-i"
	}

	tmpMount := tmpDir + ":" + dockerTmpDir
	testDataMount := testDataDir + ":" + dockerTestDataDir
	cmdArgs := []string{
		"run",
		"-e", "GOFLAGS",
		termOptions,
		"-v", tmpMount,
		"-v", testDataMount,
		dockerImage,
	}

	cmdArgs = append(cmdArgs, args...)

	return sh.RunV(dockerCmd, cmdArgs...)
}

// isExecutable checks if name is an executable in the current path.
func isExecutable(name string) bool {
	return executablePath(name) != ""
}

// executablePath returns the full path of an executable found either in the current
// path or in $GOPATH/bin. It returns an empty string if the executable is not found
// or is not executable.
func executablePath(name string) string {
	path, err := exec.LookPath(name)
	if err == nil && path != "" {
		return path
	}

	path = filepath.Join(goPathBinDir, name)
	if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
		return path
	}

	return ""
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
