package main

import (
	"fmt"
	"github.com/duyleekun/gitflow-go/shared"
	"github.com/manifoldco/promptui"
	"github.com/xanzy/go-gitlab"
	"log"
	"strings"
)
import "flag"

var nFlag = flag.String("api-token", "", "gitlab token")
var hookTokenFlag = flag.String("hook-token", "", "Webhook secret token")

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	git, err := gitlab.NewClient(*nFlag)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	v := gitlab.MaintainerPermissions
	orderBy := "last_activity_at"
	sort := "desc"
	archived := false
	projects, _, err := git.Projects.ListProjects(&gitlab.ListProjectsOptions{
		Archived:       &archived,
		MinAccessLevel: &v,
		OrderBy:        &orderBy,
		Sort:           &sort,
		ListOptions: gitlab.ListOptions{
			Page:    0,
			PerPage: 100,
		},
	})
	var projectNames []string
	for _, project := range projects {
		projectNames = append(projectNames, project.NameWithNamespace)
	}

	prompt1 := promptui.Select{
		Label: "Select Project",
		Items: projectNames,
		Searcher: func(input string, index int) bool {
			return strings.Index(projects[index].NameWithNamespace, input) >= 0
		},
		StartInSearchMode: true,
	}

	selectedProjectIndex, _, err := prompt1.Run()

	if err != nil {
		shared.PrintVerbose("Prompt failed %v\n", err)
		return
	}

	chosenProject := projects[selectedProjectIndex]
	shared.PrintVerbose("You chose %q\n", chosenProject.PathWithNamespace)
	listProtectedBranches, _, _ := git.ProtectedBranches.ListProtectedBranches(chosenProject.ID, &gitlab.ListProtectedBranchesOptions{PerPage: 100})
	for _, branch := range listProtectedBranches {
		_, err := git.ProtectedBranches.UnprotectRepositoryBranches(chosenProject.ID, branch.Name)
		shared.HandleError(err, "UnprotectRepositoryBranches %s", branch.Name)
	}

	defaultBranchName := "main"

	branches, _, err := git.Branches.ListBranches(chosenProject.ID, &gitlab.ListBranchesOptions{Search: &defaultBranchName})
	shared.HandleError(err, "ListBranches")

	mainFound := false
	for _, branch := range branches {
		if branch.Name == defaultBranchName {
			mainFound = true
		}
	}

	if !mainFound {
		oldDefaultBranch := promptRef(defaultBranchName)

		createBranch(git, chosenProject, defaultBranchName, oldDefaultBranch)

		_, _, err = git.Projects.EditProject(chosenProject.ID, &gitlab.EditProjectOptions{
			DefaultBranch: &defaultBranchName,
		})
		shared.HandleError(err, "EditProject DefaultBranch %s", defaultBranchName)
		_, err = git.Branches.DeleteBranch(chosenProject.ID, oldDefaultBranch)
		shared.HandleError(err, "DeleteBranch %s", oldDefaultBranch)
	}
	removeSourceBranchAfterMerge := true
	onlyAllowMergeIfPipelineSucceeds := true
	onlyAllowMergeIfAllDiscussionsAreResolved := false
	mergeMethod := gitlab.FastForwardMerge
	_, _, err = git.Projects.EditProject(chosenProject.ID, &gitlab.EditProjectOptions{
		DefaultBranch:                             &defaultBranchName,
		RemoveSourceBranchAfterMerge:              &removeSourceBranchAfterMerge,
		MergeMethod:                               &mergeMethod,
		OnlyAllowMergeIfPipelineSucceeds:          &onlyAllowMergeIfPipelineSucceeds,
		OnlyAllowMergeIfAllDiscussionsAreResolved: &onlyAllowMergeIfAllDiscussionsAreResolved,
	})

	shared.HandleError(err, "EditProject\n\tDefaultBranch %s\n\tRemoveSourceBranchAfterMerge %t\n\tMergeMethod %s\n\tOnlyAllowMergeIfPipelineSucceeds %t\n\tOnlyAllowMergeIfAllDiscussionsAreResolved %t",
		defaultBranchName, removeSourceBranchAfterMerge, mergeMethod, onlyAllowMergeIfPipelineSucceeds, onlyAllowMergeIfAllDiscussionsAreResolved)
	shared.PrintVerbose("Manually set 'Squash commits when merging' to 'Require' here  %s/edit", chosenProject.WebURL)
	_, err = git.Branches.DeleteMergedBranches(chosenProject.ID)
	shared.HandleError(err, "DeleteMergedBranches")

	for true {
		branchToCreate := promptBranch("env")
		ref := promptRef(branchToCreate)
		if len(branchToCreate) == 0 || len(ref) == 0 {
			break
		}
		createBranch(git, chosenProject, branchToCreate, ref)
	}

	protectBranch(git, chosenProject, "main", gitlab.NoPermissions, gitlab.DeveloperPermissions)
	protectBranch(git, chosenProject, "env/*", gitlab.NoPermissions, gitlab.MaintainerPermissions)

	setupWebhook(git, chosenProject, *hookTokenFlag, prompt("HOOK URL"))
}

func setupWebhook(git *gitlab.Client, chosenProject *gitlab.Project, hookToken string, hookURL string) {
	projectHooks, _, err := git.Projects.ListProjectHooks(chosenProject.ID, &gitlab.ListProjectHooksOptions{
		Page:    0,
		PerPage: 100,
	})
	for _, hook := range projectHooks {
		_, err := git.Projects.DeleteProjectHook(chosenProject.ID, hook.ID)
		shared.HandleError(err, "DeleteProjectHook")
	}
	trueP := true
	falseP := false
	_, _, err = git.Projects.AddProjectHook(chosenProject.ID, &gitlab.AddProjectHookOptions{
		URL:                    &hookURL,
		ConfidentialNoteEvents: &falseP,
		PushEvents:             &trueP,
		//PushEventsBranchFilter:   nil,
		IssuesEvents:             &falseP,
		ConfidentialIssuesEvents: &falseP,
		MergeRequestsEvents:      &trueP,
		TagPushEvents:            &trueP,
		NoteEvents:               &falseP,
		JobEvents:                &trueP,
		PipelineEvents:           &trueP,
		WikiPageEvents:           &falseP,
		DeploymentEvents:         &trueP,
		ReleasesEvents:           &falseP,
		EnableSSLVerification:    &trueP,
		Token:                    &hookToken,
	})
	shared.HandleError(err, "AddProjectHook %s", hookURL)
}

func createBranch(git *gitlab.Client, chosenProject *gitlab.Project, branch string, ref string) {
	_, _, err := git.Branches.CreateBranch(chosenProject.ID, &gitlab.CreateBranchOptions{
		Branch: &branch,
		Ref:    &ref,
	})
	shared.HandleError(err, "createBranch %s %s", branch, ref)
}

func protectBranch(git *gitlab.Client, project *gitlab.Project, branchNameToProtect string, push gitlab.AccessLevelValue, merge gitlab.AccessLevelValue) {
	maintainerPermission := gitlab.MaintainerPermissions
	allowedForcePush := false
	_, _, err := git.ProtectedBranches.ProtectRepositoryBranches(project.ID, &gitlab.ProtectRepositoryBranchesOptions{
		Name:                 &branchNameToProtect,
		PushAccessLevel:      &push,
		MergeAccessLevel:     &merge,
		UnprotectAccessLevel: &maintainerPermission,
		AllowForcePush:       &allowedForcePush,
	})
	shared.HandleError(err, "protectBranch %s", branchNameToProtect)
}

func promptRef(branchName string) string {
	return prompt(fmt.Sprintf("Ref for %s", branchName))
}

func promptBranch(reason string) string {
	return prompt(fmt.Sprintf("Create branch for %s", reason))
}

func prompt(promptMessage string) string {
	//validate := func(input string) error {
	//	_, err := strconv.ParseFloat(input, 64)
	//	if err != nil {
	//		return errors.New("Invalid number")
	//	}
	//	return nil
	//}
	//
	prompt := promptui.Prompt{
		Label: promptMessage,
		//Validate: validate,
	}

	result, err := prompt.Run()

	if err != nil {
		shared.PrintVerbose("Prompt failed %v\n", err)
		return ""
	}

	return result
}
