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

func getPR(rt *Runtime) *PullRequest {

	if rt.Pr == "" {
		log.Fatal("tnv.pr url is nil")
	}

	fmt.Println(rt.Pr)

	prResp, err := githubClient.Client().Get(rt.Pr)
	if prResp == nil && err != nil {
		log.Fatal(err.Error())
	}

	if rt.Debug {
		log.Println("pr retest number: ", prResp.Body)
	}

	return &PullRequest{
		Branch: "",
		Number: 12,
		Commit: "",
	}

}

func getComment(rt *Runtime) bool {

	//githubClient.Issues.ListComments(
	//	context.Background(),
	//	rt.Owner,
	//	rt.Repo,
	//	rt.Pr,
	//	nil,
	//)
	//fmt.Println(comments)

	return false

}

func retest() {

	commands := InitRetestCommands()

	fmt.Printf("%v\n", commands)

	pr := getPR(commands)

	fmt.Println(pr)

	if rerun := getComment(commands); rerun {
		fmt.Println(rerun)
	} else {
		fmt.Println("no rerun")
	}

}

func Run() {
	retest()
}
