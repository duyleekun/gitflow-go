package main

import (
	"fmt"
	"github.com/duyleekun/gitflow-go/shared"
	gitlabhook "github.com/go-playground/webhooks/v6/gitlab"
	"github.com/xanzy/go-gitlab"
	"net/http"
	"regexp"
	"strconv"
)

import "flag"

var nFlag = flag.String("api-token", "", "gitlab token")
var hookTokenFlag = flag.String("hook-token", "", "Webhook secret token")

var refExp = regexp.MustCompile(`(?:refs/heads/)?(([\w-]+)(?:/([\w-]+))?)`)

func main() {
	flag.Parse()

	git, err := gitlab.NewClient(*nFlag)
	shared.HandleError(err, "NewClient")

	hook, _ := gitlabhook.New(gitlabhook.Options.Secret(*hookTokenFlag))

	http.HandleFunc("/webhooks", func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, gitlabhook.PushEvents, gitlabhook.MergeRequestEvents)
		if err != nil {
			if err == gitlabhook.ErrEventNotFound {
				// ok event wasn;t one of the ones asked to be parsed
			}
		}
		switch payload.(type) {

		case gitlabhook.PushEventPayload:
			pushEventPayload := payload.(gitlabhook.PushEventPayload)

			match := refExp.FindStringSubmatch(pushEventPayload.Ref)
			branchName := match[1]
			branchType := match[2]
			//branchSubName := match[3]
			targetBranch := "main"

			shared.PrintVerbose("%+v", pushEventPayload)
			if pushEventPayload.Before == "0000000000000000000000000000000000000000" {
				switch branchType {
				case "feature", "hotfix":
					title := fmt.Sprintf("Draft: Merge '%s' into %s", branchName, targetBranch)
					assigneeId := int(pushEventPayload.UserID)
					createMR(git, int(pushEventPayload.ProjectID), title, branchName, targetBranch, assigneeId, true)
				}
			} else if pushEventPayload.After == "0000000000000000000000000000000000000000" {
				//BRANCH DELETED, delete MR too
				state := "opened"
				requests, _, err := git.MergeRequests.ListProjectMergeRequests(int(pushEventPayload.ProjectID), &gitlab.ListProjectMergeRequestsOptions{
					ListOptions: gitlab.ListOptions{
						Page:    0,
						PerPage: 100,
					},
					State:        &state,
					SourceBranch: &branchName,
					TargetBranch: &targetBranch,
				})
				shared.HandleError(err, "ListProjectMergeRequests")
				for _, request := range requests {
					_, err := git.MergeRequests.DeleteMergeRequest(int(pushEventPayload.ProjectID), request.IID)
					shared.PrintVerbose("%+v", request)
					shared.HandleError(err, "DeleteMergeRequest", strconv.Itoa(request.IID))
				}
			}
		case gitlabhook.MergeRequestEventPayload:
			mergeRequestEventPayload := payload.(gitlabhook.MergeRequestEventPayload)
			shared.PrintVerbose("mergeRequestEventPayload %+v", mergeRequestEventPayload)
			if mergeRequestEventPayload.ObjectKind == "merge_request" && mergeRequestEventPayload.ObjectAttributes.Action == "merge" {
				sourceMatches := refExp.FindStringSubmatch(mergeRequestEventPayload.ObjectAttributes.SourceBranch)
				sourceBranchName := sourceMatches[1]
				sourceBranchType := sourceMatches[2]
				//mergeRequestEventPayload.ObjectAttributes.Action
				targetMatches := refExp.FindStringSubmatch(mergeRequestEventPayload.ObjectAttributes.TargetBranch)
				shared.PrintVerbose("%s\n", mergeRequestEventPayload.ObjectAttributes.TargetBranch)
				//targetBranchName := targetMatches[1]
				targetBranchType := targetMatches[2]
				switch targetBranchType {
				case "main":
					search := "^env/"
					envBranches, _, err := git.Branches.ListBranches(int(mergeRequestEventPayload.Project.ID), &gitlab.ListBranchesOptions{
						ListOptions: gitlab.ListOptions{PerPage: 100},
						Search:      &search,
					})
					shared.HandleError(err, "ListBranches")

					switch sourceBranchType {
					case "hotfix":
						println("sourceBranchType hotfix")
						//	+ hotfix/ merged then
						//- auto create Cherry branch cherry/feature/<feature_name>/env/<env_name> from env/<env_name> with commits from feature/<feature_name>
						for _, envBranch := range envBranches {
							cherryBranch := fmt.Sprintf("flowcherry/%s/%s", sourceBranchName, envBranch.Name)
							createBranch(git, int(mergeRequestEventPayload.Project.ID), cherryBranch, envBranch.Name)
							commits, _, _ := git.MergeRequests.GetMergeRequestCommits(
								int(mergeRequestEventPayload.Project.ID),
								int(mergeRequestEventPayload.ObjectAttributes.IID),
								&gitlab.GetMergeRequestCommitsOptions{
									Page:    0,
									PerPage: 100,
								})
							for i, _ := range commits {
								//shared.PrintVerbose("mergeRequestEventPayload %+v", commits[len(commits)-1-i])
								_, _, err := git.Commits.CherryPickCommit(int(mergeRequestEventPayload.Project.ID), commits[len(commits)-1-i].ID, &gitlab.CherryPickCommitOptions{
									Branch: &cherryBranch,
								})
								shared.HandleError(err, "CherryPickCommit")
							}
							title := fmt.Sprintf("Draft: Merge '%s' into %s", sourceBranchName, envBranch.Name)
							assigneeId := int(mergeRequestEventPayload.ObjectAttributes.Assignee.ID)
							createMR(git, int(mergeRequestEventPayload.Project.ID), title, cherryBranch, envBranch.Name, assigneeId, true)
						}
					case "feature":
						println("sourceBranchType feature")
						//+ feature/ merged then
						//- auto create from default to all env branches (Release PR)
					}

				}

			}
		}
	})
	http.ListenAndServe(":8080", nil)
}

func createMR(git *gitlab.Client, projectID int, title string, branchName string, targetBranch string, assigneeId int, removeSourceBranch bool) {
	_, _, err := git.MergeRequests.CreateMergeRequest(projectID, &gitlab.CreateMergeRequestOptions{
		Title:              &title,
		Description:        nil,
		SourceBranch:       &branchName,
		TargetBranch:       &targetBranch,
		AssigneeID:         &assigneeId,
		RemoveSourceBranch: &removeSourceBranch,
	})
	shared.HandleError(err, "CreateMergeRequest")
}

func createBranch(git *gitlab.Client, chosenProjectID interface{}, branch string, ref string) {
	_, _, err := git.Branches.CreateBranch(chosenProjectID, &gitlab.CreateBranchOptions{
		Branch: &branch,
		Ref:    &ref,
	})
	shared.HandleError(err, "CreateBranch %s %s", branch, ref)
}
