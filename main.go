package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	otelOperatorHttpUpstreamUrl = "https://github.com/open-telemetry/opentelemetry-operator.git"
)

func main() {
	startCommit := "8e7a34e2297dbb2fe83bb7db2945636c81bf320b" // TODO get these and the repo URL from arguments
	finishCommit := "b18ddf1f4b49c422d87d394ba1d51d01ddbab68f"
	targetDirectory, err := ioutil.TempDir("/tmp", "otel-operator")
	checkIfError(err)
	defer os.RemoveAll(targetDirectory)
	repository := cloneRepository(targetDirectory)

	startTime, finishTime, err := getStartAndFinishTimeStamps(err, repository, startCommit, finishCommit)
	fmt.Println("Timestamps", startTime, finishTime)

	//  git log --ancestry-path 8e7a34e2297dbb2fe83bb7db2945636c81bf320b..b18ddf1f4b49c422d87d394ba1d51d01ddbab68f --oneline
	logOptions := &git.LogOptions{Since: &startTime, Until: &finishTime}
	commitIterator, err := repository.Log(logOptions)
	checkIfError(err)
	count := 0
	err = commitIterator.ForEach(func(commit *object.Commit) error { // FIXME we're getting 64 entries here, but 63 online; there could be an off by one problem
		count += 1
		fileIterator, err := commit.Files() // TODO: Look at TREE also
		checkIfError(err)
		fileCount := 0
		fileIterator.ForEach(func(f *object.File) error { // TODO what if there is more than one change?
			//if strings.HasSuffix(f.Name, "Dockerfile") {
			//	fmt.Println(">>>>>>", commit.Hash, f.Name)
			//}
			fileCount += 1
			return nil // end of fileIterator
		})
		fmt.Println(commit.Hash, "has", fileCount)
		return nil // End of commit iterator
	})
	fmt.Println("Got", count, "commits")
}

func getStartAndFinishTimeStamps(err error, repository *git.Repository, startCommit string, finishCommit string) (time.Time, time.Time, error) {
	startCommitTime := time.Now()
	finishCommitTime := time.Now()
	commitIter, err := repository.CommitObjects()
	checkIfError(err)

	defer commitIter.Close()

	startCommitFound := false
	finishCommitFound := false

	err = commitIter.ForEach(func(commit *object.Commit) error {
		//fmt.Println(startCommitTime, finishCommitTime)
		if !startCommitFound && commit.Hash.String() == startCommit {
			startCommitFound = true
			startCommitTime = commit.Author.When
			fmt.Println("Start commit found")
		}
		if !finishCommitFound && commit.Hash.String() == finishCommit {
			finishCommitFound = true
			finishCommitTime = commit.Author.When
			fmt.Println("Finish commit found")
		}
		// is there any way to quit when both values are found?
		return nil
	})

	// FIXME return an error here if both have not been found
	if err != nil {
		fmt.Println("Error:", err)
		return startCommitTime, finishCommitTime, err
	}

	return startCommitTime, finishCommitTime, nil

}

func cloneRepository(targetDirectory string) *git.Repository {
	os.RemoveAll(targetDirectory) // ignore the error here, at least if the directory doesn't exist...
	cloneOptions := &git.CloneOptions{URL: otelOperatorHttpUpstreamUrl, Progress: os.Stdout}
	repository, err := git.PlainClone(targetDirectory, false, cloneOptions)
	checkIfError(err)
	return repository
}

// checkIfError should be used to naively panics if an error is not nil.  Stolen from https://github.com/go-git/go-git/blob/master/_examples/common.go
func checkIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	debug.PrintStack()
	os.Exit(1)
}
