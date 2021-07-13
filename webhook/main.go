package main

import (
	"fmt"
	"github.com/duyleekun/gitflow-go/shared"
	gitlabhook "github.com/go-playground/webhooks/v6/gitlab"
	"github.com/google/shlex"
	"github.com/xanzy/go-gitlab"
	"net/http"
	"regexp"
	"strconv"
)

import "flag"

var nFlag = flag.String("api-token", "", "gitlab token")
var hookTokenFlag = flag.String("hook-token", "", "Webhook secret token")

var defaultProtectedBranchExp = regexp.MustCompile(`^(?:main|env/.+)$`)
var envBranchExp = regexp.MustCompile(`^env/.+$`)
var refExp = regexp.MustCompile(`(?:refs/heads/)?(([\w-]+)(?:/([\w-]+))?)`)
var commandExp = regexp.MustCompile(`\$(.*)`)

func main() {
	flag.Parse()

	git, err := gitlab.NewClient(*nFlag)
	shared.HandleError(err, "NewClient")

	hook, _ := gitlabhook.New(gitlabhook.Options.Secret(*hookTokenFlag))

	http.HandleFunc("/webhooks", func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, gitlabhook.PushEvents, gitlabhook.MergeRequestEvents, gitlabhook.CommentEvents)
		if err != nil {
			if err == gitlabhook.ErrEventNotFound {
				// ok event wasn;t one of the ones asked to be parsed
			}
		}
		switch payload.(type) {
		case gitlabhook.CommentEventPayload:
			commentEventPayload := payload.(gitlabhook.CommentEventPayload)
			submatch := commandExp.FindStringSubmatch(commentEventPayload.ObjectAttributes.Description)
			if len(submatch) > 0 {
				split, err := shlex.Split(submatch[1])
				shared.HandleError(err, "shlex.Split %s", submatch)
				chosenProjectId := int(commentEventPayload.ProjectID)
				//This command only works within PR
				if commentEventPayload.MergeRequest.IID > 0 {
					go func() {
						switch split[0] {
						case "force_merge":
							if !isEnvBranch(commentEventPayload.MergeRequest.TargetBranch) {
								shared.ReplyComment(git, commentEventPayload, "Force deploy failed, TargetBranch %s doesn't match %s", commentEventPayload.MergeRequest.SourceBranch, envBranchExp.String())
								break
							}

							if !isDefaultProtectedBranch(commentEventPayload.MergeRequest.SourceBranch) {
								shared.ReplyComment(git, commentEventPayload, "Force deploy failed, source %s doesn't match %s", commentEventPayload.MergeRequest.SourceBranch, defaultProtectedBranchExp.String())
								break
							}

							// Delete target branch
							shared.ReplyComment(git, commentEventPayload, "I'm about to delete %s", commentEventPayload.MergeRequest.TargetBranch)
							shared.DeleteBranch(git, chosenProjectId, commentEventPayload.MergeRequest.TargetBranch)

							// Recreate target branch from source branch
							shared.ReplyComment(git, commentEventPayload, "I'm about to create %s from %s", commentEventPayload.MergeRequest.TargetBranch, commentEventPayload.MergeRequest.SourceBranch)
							shared.CreateBranch(git, chosenProjectId, commentEventPayload.MergeRequest.TargetBranch, commentEventPayload.MergeRequest.SourceBranch)

							// Close PR
							shared.ReplyComment(git, commentEventPayload, "Force deploy successful, I'm gonna close this PR")
							shared.ClosePR(git, chosenProjectId, int(commentEventPayload.MergeRequest.IID))
						}
					}()
				}
			}
		case gitlabhook.PushEventPayload:
			pushEventPayload := payload.(gitlabhook.PushEventPayload)

			match := refExp.FindStringSubmatch(pushEventPayload.Ref)
			branchName := match[1]
			branchType := match[2]
			//branchSubName := match[3]
			targetBranch := "main"

			if pushEventPayload.Before == "0000000000000000000000000000000000000000" {
				switch branchType {
				case "feature", "hotfix":
					title := fmt.Sprintf("Draft: Merge '%s' into %s", branchName, targetBranch)
					assigneeId := int(pushEventPayload.UserID)
					shared.CreateMR(git, int(pushEventPayload.ProjectID), title, branchName, targetBranch, assigneeId, true)
				}
			} else if pushEventPayload.After == "0000000000000000000000000000000000000000" {
				//BRANCH DELETED, delete MR too
				state := "opened"
				mergeRequests, _, err := git.MergeRequests.ListProjectMergeRequests(int(pushEventPayload.ProjectID), &gitlab.ListProjectMergeRequestsOptions{
					ListOptions: gitlab.ListOptions{
						Page:    0,
						PerPage: 100,
					},
					State:        &state,
					SourceBranch: &branchName,
					TargetBranch: &targetBranch,
				})
				shared.HandleError(err, "ListProjectMergeRequests")
				for _, request := range mergeRequests {
					_, err := git.MergeRequests.DeleteMergeRequest(int(pushEventPayload.ProjectID), request.IID)
					shared.PrintVerbose("%+v", request)
					shared.HandleError(err, "DeleteMergeRequest", strconv.Itoa(request.IID))
				}
			}
		case gitlabhook.MergeRequestEventPayload:
			mergeRequestEventPayload := payload.(gitlabhook.MergeRequestEventPayload)
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
						go func() {
							//TODO Maybe choose the squashed one
							currentMR := GetMR(git, mergeRequestEventPayload)

							//shared.PrintVerbose("GetMergeRequest %v", currentMR)

							shaToCherryPick := currentMR.SquashCommitSHA
							if len(shaToCherryPick) == 0 {
								//Single commit MR
								shaToCherryPick = currentMR.SHA
							}

							for _, envBranch := range envBranches {
								cherryBranch := fmt.Sprintf("flowcherry/%s/%s", sourceBranchName, envBranch.Name)
								shared.ReplyMergeRequest(git, mergeRequestEventPayload, "Branch %s created", cherryBranch)
								shared.CreateBranch(git, int(mergeRequestEventPayload.Project.ID), cherryBranch, envBranch.Name)

								title := fmt.Sprintf("Draft: Merge '%s' into %s", sourceBranchName, envBranch.Name)
								assigneeId := int(mergeRequestEventPayload.ObjectAttributes.Assignee.ID)
								newlyCreatedMR := shared.CreateMR(git, int(mergeRequestEventPayload.Project.ID), title, cherryBranch, envBranch.Name, assigneeId, true)
								shared.ReplyMergeRequest(git, mergeRequestEventPayload, "MR !%d created", newlyCreatedMR.IID)

								_, _, err := git.Commits.CherryPickCommit(int(mergeRequestEventPayload.Project.ID), shaToCherryPick, &gitlab.CherryPickCommitOptions{
									Branch: &cherryBranch,
								})
								shared.HandleIgnoreError(err, "CherryPickCommit %s %s", shaToCherryPick, cherryBranch)
								shared.ReplyMergeRequest(git, mergeRequestEventPayload, "Cherry picked from %s to %s", shaToCherryPick, cherryBranch)
							}
						}()
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

func isEnvBranch(branch string) bool {
	return envBranchExp.MatchString(branch)
}

func isDefaultProtectedBranch(branch string) bool {
	return defaultProtectedBranchExp.MatchString(branch)
}

func GetMR(git *gitlab.Client, mergeRequestEventPayload gitlabhook.MergeRequestEventPayload) *gitlab.MergeRequest {
	includeRebaseInProgress := true
	mr, _, err := git.MergeRequests.GetMergeRequest(
		int(mergeRequestEventPayload.Project.ID),
		int(mergeRequestEventPayload.ObjectAttributes.IID), &gitlab.GetMergeRequestsOptions{
			//RenderHTML:                  nil,
			//IncludeDivergedCommitsCount: nil,
			IncludeRebaseInProgress: &includeRebaseInProgress,
		},
	)
	shared.HandleError(err, "GetMergeRequest %d %d", int(mergeRequestEventPayload.Project.ID), int(mergeRequestEventPayload.ObjectAttributes.IID))
	return mr
}
