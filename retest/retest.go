package retest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/actions-go/toolkit/core"
	"github.com/actions-go/toolkit/github"
	github2 "github.com/google/go-github/v42/github"
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
		owner = strings.Split(nwo, "/")[0]
		repo = strings.Split(nwo, "/")[1]
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

func getSuffix(pr string) int {

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
		getSuffix(rt.Pr),
	)

	if pr == nil && prResp.StatusCode != 200 && prResp.StatusCode != 201 && err != nil {

		log.Fatal("pr not found, err: ", err)
	}

	return &PullRequest{
		Branch: pr.Head.GetRef(),
		Number: pr.GetNumber(),
		Commit: pr.Head.GetSHA(),
	}

}

func addReaction(rt *Runtime, content string) bool {

	_, response, err := githubClient.Reactions.CreateIssueCommentReaction(
		context.Background(),
		rt.Owner,
		rt.Repo,
		int64(rt.Comment),
		content,
	)

	if response.StatusCode != 200 && response.StatusCode != 201 && err != nil {

		log.Fatal("failed to add reaction, error: ", err)
		return false
	}

	return true

}

func getFailedJos(rt *Runtime, pr *PullRequest) (failedChecks []*GHRetest) {

	ref, response, err := githubClient.Checks.ListCheckRunsForRef(
		context.Background(),
		rt.Owner,
		rt.Repo,
		pr.Commit,
		nil,
	)

	if response.StatusCode != 200 && response.StatusCode != 201 && err != nil {

		log.Fatal("failed to get check runs, error: ", err)

	}

	for order, run := range ref.CheckRuns {

		if rt.Debug {
			log.Printf("check run: %v, order: %v\n", run, order)
		}

		if run.GetConclusion() == "failure" {

			failedChecks = append(failedChecks, &GHRetest{
				Name: run.GetName(),
				Url:  run.GetDetailsURL(),
			})
		}

	}

	return failedChecks
}

func rerunJobs(rt *Runtime, failedJobs []*GHRetest) (result *GHRetestResult) {

	var errorNum int

	for _, job := range failedJobs {

		fmt.Printf("retesting check: %v\n %v\n", job.Name, job.Url)

		u := fmt.Sprintf("repos/%v/%v/actions/jobs/%v/rerun", rt.Owner, rt.Repo, getSuffix(job.Url))
		req, err := githubClient.NewRequest(http.MethodPost, u, nil)

		if err != nil {
			log.Fatal("failed to create request, error: ", err)
			return nil
		}

		job := new(github2.WorkflowJob)
		resp, err := githubClient.Do(context.Background(), req, job)

		if resp.StatusCode != 201 && err != nil {

			errorNum++
			log.Fatal("failed to retest job, error: ", err)
			return nil
		}

	}

	return &GHRetestResult{
		Error:    errorNum,
		Retested: len(failedJobs),
	}
}

func retest() {

	rt := InitRetestCommands()
	pr := getPR(rt)
	failedJosList := getFailedJos(rt, pr)

	if len(failedJosList) == 0 {

		log.Println("no failed checks found")
		return
	}

	if rt.Debug {
		log.Printf("Runtime info: %v\n: ", rt)
		log.Printf("pr info: %v", pr)
	}

	result := rerunJobs(rt, failedJosList)
	if result.Error != 0 {

		addReaction(rt, "-1")
	}
	if result.Error == 0 {

		log.Println("all checks have been restarted")
		addReaction(rt, "rocket")
	} else {

		log.Printf("failed to restart some checks, error times: %v\n", result.Error)
		addReaction(rt, "confused")
	}

}

func Run() {

	defer func() {
		if err := recover(); err != nil {

			log.Println("retest error: ", err)
			core.SetFailedf("Retest action failure, error is %v\n", err)
		}
	}()

	retest()
}
