package main

/*
From: https://stackoverflow.com/questions/34018908/golang-why-dont-we-have-a-set-datastructure

The usual answer for golang question: "Why provide a feature when you can rewrite it in just a few lines?". This is why
something that can be done in 3 explicit lines in python (or many other languages) takes 50+ obscure lines in go. This
is one of the reasons (along with single letter variables) why I hate reading go code. It's uselessly long, just doing
with for loops what should be done by a clear, efficient and well tested properly named function. Go "spirit" is just
throwing away 50 years of good software engineering practice with dubious justifications. –
Colin Pitrat
 Jul 8, 2021 at 11:34
*/
import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	startCommitHashString       = "8e7a34e2297dbb2fe83bb7db2945636c81bf320b" // TODO get these and the repo URL from arguments.
	finishCommitHashString      = "b18ddf1f4b49c422d87d394ba1d51d01ddbab68f"
	otelOperatorHttpUpstreamUrl = "https://github.com/open-telemetry/opentelemetry-operator.git"
)

func main() {
	// TODO add the list of files we care about...

	// Get the repo plus the name of the directory it was cloned to.  The latter is so we can clean up afterwards.
	repository, targetDirectory := cloneRepository()
	defer os.RemoveAll(targetDirectory)

	// Get the start and end commits, then the names of all files changed between the two of them (inclusive)
	startCommit, finishCommit := getStartAndFinishCommits(repository, startCommitHashString, finishCommitHashString)
	sortedFileNames := getChangedFileNames(startCommit, finishCommit)

	fmt.Println("-------------------------------------------------------------------------")
	sort.Strings(sortedFileNames)
	for _, fileName := range sortedFileNames {
		fmt.Println(fileName)
	}
}

func getStartAndFinishCommits(repository *git.Repository, startCommitHashString string, finishCommitHashString string) (*object.Commit, *object.Commit) {
	startCommit, err := repository.CommitObject(plumbing.NewHash(startCommitHashString))
	checkIfError(err)
	finishCommit, err := repository.CommitObject(plumbing.NewHash(finishCommitHashString))
	checkIfError(err)
	return startCommit, finishCommit
}

func getChangedFileNames(startCommit *object.Commit, finishCommit *object.Commit) []string {
	patch, err := startCommit.Patch(finishCommit)
	checkIfError(err)
	changedFileNames := make(map[string]bool) // We really want a set here, but the fucking geniuses who created go didn't provide one
	filePatches := patch.FilePatches()
	for _, filePatch := range filePatches { // I *think* we are just care about to.  If there is just a from, the file was deleted.  Just a to means a file was added
		// I *think* we are just care about to.  If there is just a from, the file was deleted, just a to means a file was added
		_, to := filePatch.Files()
		if to != nil { // TODO && to.Path() is in the list of files we care about
			changedFileNames[to.Path()] = true
		}
	}

	sortedFileNames := []string{}
	for fileName, _ := range changedFileNames {
		sortedFileNames = append(sortedFileNames, fileName)
	}
	return sortedFileNames
}

func cloneRepository() (*git.Repository, string) {
	targetDirectory, err := ioutil.TempDir("/tmp", "otel-operator")
	os.RemoveAll(targetDirectory) // ignore the error here, at least if the directory doesn't exist...
	cloneOptions := &git.CloneOptions{URL: otelOperatorHttpUpstreamUrl, Progress: os.Stdout}
	repository, err := git.PlainClone(targetDirectory, false, cloneOptions)
	checkIfError(err)

	repository.CommitObjects()

	fmt.Println("Target directory name is", targetDirectory)
	return repository, targetDirectory
}

// checkIfError should be used to naively panic if an error is not nil.  Stolen from https://github.com/go-git/go-git/blob/master/_examples/common.go
func checkIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	debug.PrintStack()
	os.Exit(1)
}
