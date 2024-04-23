package retest

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/actions-go/toolkit/core"
	"github.com/actions-go/toolkit/github"
)

var (
	githubClient = github.NewClient()
)

func InitRetestCommands() *Runtime {

	commentInput, _ := core.GetInput("comment-id")
	comment, _ := strconv.Atoi(commentInput)
	pr, _ := core.GetInput("pr-url")
	nwo := os.Getenv("GITHUB_REPOSITORY")
	debug := os.Getenv("CI_DEBUG") != "" && os.Getenv("CI_DEBUG") != "false"
	var repo, owner string

	if nwo != "" {
		repo = strings.Split(nwo, "/")[0]
		owner = strings.Split(nwo, "/")[1]
	} else {
		log.Fatal("GITHUB_REPOSITORY must not be nil")
	}

	return &Runtime{
		Pr:      pr,
		Comment: comment,
		Repo:    repo,
		Owner:   owner,
		Debug:   debug,
	}
}

func getComment() bool {

	comments := github.Context.Payload.PullRequest.Comments
	fmt.Println(comments)

	return false

}

func retest() {

	if rerun := getComment(); rerun {
		fmt.Println(rerun)
	} else {
		fmt.Println("no rerun")
	}

}

func Run() {
	retest()
}
