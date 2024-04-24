package retest

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
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

		log.Fatal("pr not found, err: ", err)
	}

	if rt.Debug {

		log.Println("pr retest number: ", prResp.Body)
	}

	return &PullRequest{
		Branch: pr.Head.GetRef(),
		Number: pr.GetNumber(),
		Commit: pr.Head.GetSHA(),
	}

}

func addReaction(rt *Runtime, content string) bool {

	_, response, err := githubClient.Reactions.CreateCommentReaction(
		context.Background(),
		rt.Owner,
		rt.Repo,
		int64(rt.Comment),
		content,
	)

	if response.StatusCode != 200 && err != nil {

		log.Fatal("failed to add reaction, error: ", err)
		return false
	}

	return true

}

func getRetestActionTask(rt *Runtime, pr *PullRequest) (failedChecks []*GHRetest) {

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

	for _, check := range ref.CheckRuns {

		if check.GetExternalID() == "" {

			continue
		}

		if rt.Debug {

			log.Printf("rerun failed checks %v %v %v\n", *check.Name, *check.Conclusion, *check.ExternalID)
		}

		if check.GetConclusion() == "failure" ||
			check.GetConclusion() == "timed_out" ||
			check.GetConclusion() == "cancelled" {

			if check.GetName() == "" {

				*check.Name = "unknown"
			}

			failedChecks = append(failedChecks, &GHRetest{
				Name: check.GetName(),
				Url: fmt.Sprintf("/repos/%s/%s/actions/runs/%s/rerun-failed-jobs",
					rt.Owner,
					rt.Repo,
					check.GetExternalID(),
				),
			})

			// Create a new check from old failed check.
			toDelete := []string{
				"PullRequests",
				"App",
				"CheckSuite",
				"Conclusion",
				"NodeId",
				"StartedAt",
				"CompletedAt",
				"Id",
			}

			checkRunsFiledReload(check, toDelete...)
			checkRunsFiledReload(check, "url")
			checkRunsFiledReload(check.GetOutput(), "annotation")

			*check.GetOutput().Title = strings.Replace(check.GetOutput().GetTitle(), "failed", "restarted", 1)
			lines := strings.Split(check.GetOutput().GetText(), "\n")
			line0 := strings.Replace(lines[0], "Check run finished (failure :x:)", "Check run restarted", 1)
			*check.GetOutput().Text = fmt.Sprintf("%s\n%s", line0, strings.Join(lines[1:], "\n"))
			*check.GetOutput().Summary = "Check is running again"
			*check.Status = "in_progress"

			fmt.Printf("update check: %v\n", check)

			failedChecks = append(failedChecks, &GHRetest{

				Name:   check.GetName(),
				Url:    fmt.Sprintf("/repos/%s/%s/check-runs", rt.Owner, rt.Repo),
				Config: check,
			})
		}

	}

	return failedChecks
}

func checkRunsFiledReload(obj interface{}, toDelete ...string) {

	if len(toDelete) == 1 {

		deleteField(obj, toDelete[0])
	} else {

		t := reflect.TypeOf(obj).Elem()
		fieldNames := make([]string, 0)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldNames = append(fieldNames, field.Name)
		}

		for _, fn := range fieldNames {
			for _, str := range toDelete {
				if fn == str {
					deleteField(obj, fn)
				}
			}
		}
	}

}

func deleteField(obj interface{}, fieldName string) {

	v := reflect.ValueOf(obj).Elem()
	f := v.FieldByName(fieldName)

	if f.IsValid() && f.CanSet() {

		f.Set(reflect.Zero(f.Type()))
	}
}

func retestRuns(pr *PullRequest, rt *Runtime, failedChecks []*GHRetest) (result *GHRetestResult) {

	var errorNum int

	for _, failedCheck := range failedChecks {

		if strings.HasPrefix(failedCheck.Url, "rerun-failed-jobs") {
			log.Printf("retesting failed jobs (pr #{%d}): %v\n", pr.Number, failedCheck.Name)
		} else {
			log.Printf("restarting check (pr #{%d}): %v\n", pr.Number, failedCheck.Name)
		}

		if failedCheck.Config != nil {

			rerun, response, err := githubClient.Checks.CreateCheckRun(
				context.Background(),
				rt.Owner,
				rt.Repo,
				github2.CreateCheckRunOptions{
					Output: &github2.CheckRunOutput{
						Text:    failedCheck.Config.(*github2.CheckRun).Output.Text,
						Title:   failedCheck.Config.(*github2.CheckRun).Output.Title,
						Summary: failedCheck.Config.(*github2.CheckRun).Output.Summary,
					},
					Status: failedCheck.Config.(*github2.CheckRun).Status,
				},
			)

			if (response.StatusCode == 200 || response.StatusCode == 201) && err != nil {

				if strings.HasPrefix(failedCheck.Url, "rerun-failed-jobs") {

					fmt.Printf("::notice::Retry success: (%s)\n", failedCheck.Name)
				} else {

					fmt.Printf("::notice::Check restarted: (%s)\n %s\n", failedCheck.Name, rerun.HTMLURL)
				}
			} else {

				if strings.HasPrefix(failedCheck.Url, "rerun-failed-jobs") {

					core.Errorf("Retry failed: (%s) ... %v\n", failedCheck.Name, response.Status)
				} else {

					core.Errorf("Failed restarting check: %s\n", failedCheck.Name)
				}

				// error times ++
				errorNum++
			}
		}

	}

	return &GHRetestResult{
		Error:    errorNum,
		Retested: len(failedChecks),
	}
}

func retest() {

	rt := InitRetestCommands()
	pr := getPR(rt)
	failedCheckList := getRetestActionTask(rt, pr)

	if len(failedCheckList) == 0 {

		log.Println("no failed checks found")
		return
	}

	if rt.Debug {
		log.Printf("Runtime info: %v\n: ", rt)
		log.Printf("pr info: %v", pr)
	}

	result := retestRuns(pr, rt, failedCheckList)
	if result.Error != 0 {

		addReaction(rt, "-1")
	}
	if result.Error == 0 {

		log.Println("all checks have been restarted")
		addReaction(rt, "rocket")
	} else {

		log.Println("failed to restart some checks")
		addReaction(rt, "confused")
	}

}

func Run() {

	//defer func() {
	//	if err := recover(); err != nil {
	//
	//		log.Println("retest error: ", err)
	//		core.SetFailedf("Retest action failure, error is ", err)
	//	}
	//}()

	retest()
}
