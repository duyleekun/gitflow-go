package main

import (
	"fmt"
	"github.com/duyleekun/gitflow-go/shared"
	"github.com/xanzy/go-gitlab"
	"strings"
)
import "flag"

var nFlag = flag.String("api-token", "", "gitlab token")
var hookTokenFlag = flag.String("hook-token", "", "Webhook secret token")

func main() {
	flag.Parse()

	git, err := gitlab.NewClient(*nFlag)
	shared.HandleError(err, "NewClient")

	chosenProject := chooseProject(git)
	deleteAllProtectedBranches(git, chosenProject)

	defaultBranchName := "main"
	// Create `main` branch
	createMainBranch(git, chosenProject, defaultBranchName)

	// Update project setting
	updateProjectSetting(git, chosenProject, defaultBranchName)

	// Create ENV branch
	createEnvBranches(git, chosenProject)

	// Protect branches
	protectBranch(git, chosenProject, "main", gitlab.NoPermissions, gitlab.DeveloperPermissions)
	protectBranch(git, chosenProject, "env/*", gitlab.NoPermissions, gitlab.MaintainerPermissions)

	// Setup webhook
	setupWebhook(git, chosenProject, *hookTokenFlag, shared.PromptString("HOOK URL"))
}

func createEnvBranches(git *gitlab.Client, chosenProject *gitlab.Project) {
	for branchToCreate := promptBranch("env"); len(branchToCreate) > 0; {
		ref := promptRef(branchToCreate)
		if len(branchToCreate) == 0 || len(ref) == 0 {
			break
		}
		createBranch(git, chosenProject, branchToCreate, ref)
	}
}

func createMainBranch(git *gitlab.Client, chosenProject *gitlab.Project, defaultBranchName string) {
	if findBranch(git, chosenProject, defaultBranchName) {
		oldDefaultBranch := promptRef(defaultBranchName)

		// create new default branch
		createBranch(git, chosenProject, defaultBranchName, oldDefaultBranch)

		// update new default branch
		_, _, err := git.Projects.EditProject(chosenProject.ID, &gitlab.EditProjectOptions{
			DefaultBranch: &defaultBranchName,
		})
		shared.HandleError(err, "EditProject DefaultBranch %s", defaultBranchName)

		//delete old default branch
		_, err = git.Branches.DeleteBranch(chosenProject.ID, oldDefaultBranch)
		shared.HandleError(err, "DeleteBranch %s", oldDefaultBranch)
	}
}

func updateProjectSetting(git *gitlab.Client, chosenProject *gitlab.Project, defaultBranchName string) {
	removeSourceBranchAfterMerge := true
	onlyAllowMergeIfPipelineSucceeds := false
	onlyAllowMergeIfAllDiscussionsAreResolved := true
	mergeMethod := gitlab.FastForwardMerge
	_, _, err := git.Projects.EditProject(chosenProject.ID, &gitlab.EditProjectOptions{
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
}

func findBranch(git *gitlab.Client, chosenProject *gitlab.Project, branchName string) bool {
	branches, _, err := git.Branches.ListBranches(chosenProject.ID, &gitlab.ListBranchesOptions{Search: &branchName})
	shared.HandleError(err, "ListBranches")

	mainFound := false
	for _, branch := range branches {
		if branch.Name == branchName {
			mainFound = true
		}
	}
	return mainFound
}

func deleteAllProtectedBranches(git *gitlab.Client, chosenProject *gitlab.Project) {
	listProtectedBranches, _, _ := git.ProtectedBranches.ListProtectedBranches(chosenProject.ID, &gitlab.ListProtectedBranchesOptions{PerPage: 100})
	for _, branch := range listProtectedBranches {
		_, err := git.ProtectedBranches.UnprotectRepositoryBranches(chosenProject.ID, branch.Name)
		shared.HandleError(err, "UnprotectRepositoryBranches %s", branch.Name)
	}
}

func chooseProject(git *gitlab.Client) *gitlab.Project {
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
	shared.HandleError(err, "ListProjects")

	selectedProjectIndex := shared.PromptSelect("Select Project", len(projects), func(input string, index int) bool {
		return strings.Index(projects[index].NameWithNamespace, input) >= 0
	}, func(i int) string {
		return projects[i].PathWithNamespace
	})

	chosenProject := projects[selectedProjectIndex]
	shared.PrintVerbose("You chose %q\n", chosenProject.PathWithNamespace)
	return chosenProject
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
	return shared.PromptString(fmt.Sprintf("Ref for %s", branchName))
}

func promptBranch(reason string) string {
	return shared.PromptString(fmt.Sprintf("Create branch for %s", reason))
}
