package shared

import (
	"fmt"
	gitlabhook "github.com/go-playground/webhooks/v6/gitlab"
	"github.com/xanzy/go-gitlab"
)

func ReplyComment(git *gitlab.Client, commentEventPayload gitlabhook.CommentEventPayload, commentBodyFormat string, args ...interface{}) {
	commentBody := fmt.Sprintf(commentBodyFormat, args...)
	_, _, err := git.Notes.CreateMergeRequestNote(int(commentEventPayload.ProjectID), int(commentEventPayload.MergeRequest.IID), &gitlab.CreateMergeRequestNoteOptions{Body: &commentBody})
	HandleIgnoreError(err, "CreateMergeRequestNote %d %d %s", int(commentEventPayload.ProjectID), int(commentEventPayload.MergeRequest.IID), commentBody)
}

func ReplyMergeRequest(git *gitlab.Client, mergeRequestEventPayload gitlabhook.MergeRequestEventPayload, commentBodyFormat string, args ...interface{}) {
	commentBody := fmt.Sprintf(commentBodyFormat, args...)
	_, _, err := git.Notes.CreateMergeRequestNote(int(mergeRequestEventPayload.Project.ID), int(mergeRequestEventPayload.ObjectAttributes.IID), &gitlab.CreateMergeRequestNoteOptions{Body: &commentBody})
	HandleIgnoreError(err, "CreateMergeRequestNote %d %d %s", int(mergeRequestEventPayload.Project.ID), int(mergeRequestEventPayload.ObjectAttributes.IID), commentBody)
}

func CreateMR(git *gitlab.Client, projectID int, title string, sourceBranch string, targetBranch string, assigneeId int, removeSourceBranch bool) *gitlab.MergeRequest {
	mr, _, err := git.MergeRequests.CreateMergeRequest(projectID, &gitlab.CreateMergeRequestOptions{
		Title:              &title,
		Description:        nil,
		SourceBranch:       &sourceBranch,
		TargetBranch:       &targetBranch,
		AssigneeID:         &assigneeId,
		RemoveSourceBranch: &removeSourceBranch,
	})
	HandleError(err, "CreateMergeRequest")
	return mr
}

func CreateBranch(git *gitlab.Client, chosenProjectID int, branch string, ref string) {
	_, _, err := git.Branches.CreateBranch(chosenProjectID, &gitlab.CreateBranchOptions{
		Branch: &branch,
		Ref:    &ref,
	})
	HandleError(err, "CreateBranch %s %s", branch, ref)
}

func DeleteBranch(git *gitlab.Client, chosenProjectID interface{}, branch string) {
	_, err := git.Branches.DeleteBranch(chosenProjectID, branch)
	HandleError(err, "CreateBranch %s", branch)
}

func FindBranch(git *gitlab.Client, chosenProjectID int, branchName string) bool {
	branches, _, err := git.Branches.ListBranches(chosenProjectID, &gitlab.ListBranchesOptions{Search: &branchName})
	HandleError(err, "ListBranches")

	mainFound := false
	for _, branch := range branches {
		if branch.Name == branchName {
			mainFound = true
		}
	}
	return mainFound
}

func ClosePR(client *gitlab.Client, chosenProjectID int, chosenPRIID int) {
	stateEvent := "close"
	client.MergeRequests.UpdateMergeRequest(chosenProjectID, chosenPRIID, &gitlab.UpdateMergeRequestOptions{
		//Title:              nil,
		//Description:        nil,
		//TargetBranch:       nil,
		//AssigneeID:         nil,
		//AssigneeIDs:        nil,
		//ReviewerIDs:        nil,
		//Labels:             nil,
		//AddLabels:          nil,
		//RemoveLabels:       nil,
		//MilestoneID:        nil,
		StateEvent: &stateEvent,
		//RemoveSourceBranch: nil,
		//Squash:             nil,
		//DiscussionLocked:   nil,
		//AllowCollaboration: nil,
	})
}
