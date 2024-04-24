package retest

import (
	"context"
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
		Nwo:     nwo,
		Owner:   owner,
		Debug:   debug,
	}

}

func getPRNumber(pr string) int {

	prSplit := strings.Split(pr, "/")
	prNumber, _ := strconv.Atoi(prSplit[len(prSplit)-1])

	return prNumber
}

func getPR(rt *Runtime) *PullRequest {

	if rt.Pr == "" {
		log.Fatal("env.pr url is nil")
	}

	pr, prResp, err := githubClient.PullRequests.Get(
		context.Background(),
		rt.Owner,
		rt.Repo,
		getPRNumber(rt.Pr),
	)

	if pr == nil && prResp.StatusCode != 200 && err != nil {

		log.Fatal("pr not found")
	}

	if rt.Debug {

		log.Println("pr retest number: ", prResp.Body)
	}

	return &PullRequest{
		Branch: *pr.Head.Ref,
		Number: *pr.Number,
		Commit: *pr.Head.SHA,
	}

}

func addReaction(rt *Runtime) bool {

	_, response, err := githubClient.Reactions.CreateCommentReaction(
		context.Background(),
		rt.Owner,
		rt.Repo,
		int64(rt.Comment),
		"rocket",
	)

	if response.StatusCode != 200 && err != nil {

		log.Fatal("failed to add reaction")
		return false
	}

	return true

}

func getRetestActionTask(rt *Runtime, pr *PullRequest) (retestTasks []*GHRetest) {

	ref, response, err := githubClient.Checks.ListCheckRunsForRef(
		context.Background(),
		rt.Owner,
		rt.Repo,
		pr.Commit,
		nil,
	)

	if response.StatusCode != 200 && err != nil {

		log.Fatal("failed to get check runs")
	}

	var checkIds []*string
	for _, check := range ref.CheckRuns {

		fmt.Printf("%v\n", check)

		if check.ExternalID == nil {

			continue
		}

		checkIds = append(checkIds, check.ExternalID)
	}

	retestTasks = append(retestTasks, &GHRetest{})

	return nil
}

func retest() {

	commands := InitRetestCommands()
	pr := getPR(commands)

	if commands.Debug {
		log.Printf("commands runtime info: %v\n: ", commands)
		log.Printf("pr info: %v", pr)
	}

}

func Run() {

	retest()
}
