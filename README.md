# gitflow-go

Automated repetitive setup actions and git actions

## CLI

Developer can skip this section, this CLI is just for setup Gitflow for new project

1. Choose project (list with maintainers permission)

- Delete all protected branch options

2. Select old default branch (usually it's `develop` or `master`)

- Create `main` branch
- Set `main` branch as default
- Delete old default branch

3. Update project settings

```yaml
DefaultBranch:                             main
RemoveSourceBranchAfterMerge:              true
MergeMethod:                               FastForwardMerge
OnlyAllowMergeIfPipelineSucceeds:          false
OnlyAllowMergeIfAllDiscussionsAreResolved: true
```

Will prompt to setup the squash setting vie GUI. Gitlab API doesn't allow setting this via API yet https://gitlab.com/gitlab-org/gitlab/-/issues/333945

4. create `env/*` branches (enter twice to skip creating)

- Branch Name
- Branch Ref 

5. Protect branch

- protectBranch `main`  with `NoPermissions` for push and `DeveloperPermissions`  for merge
- protectBranch `env/*` with `NoPermissions` for push and `MaintainerPermissions` for merge

6. Setup webhook

- **[DANGER]** Delete all other hooks
- Add new webhook

## Webhook

### Events with Automation

#### On `feature/<feature_name>` or `hotfix/<fix_name>` pushed

- Create PR

```yaml
  Title:              "Draft: Merge 'feature/<feature_name> or hotfix/<fix_name>' into main",
  Description:        "TODO",
  SourceBranch:       "feature/<feature_name> or hotfix/<fix_name>",
  TargetBranch:       "main",
  AssigneeID:         "pusher id",
  RemoveSourceBranch: true,
```

#### On `feature/<feature_name>` merged to `main`
- **[RFC/TODO]** auto create PR from `main` to all `env/<stage0_env_name>` branches


#### On `hotfix/<fix_name>` merged to `main`

- For each `env/<env_name>` branch 

    - Auto create `cherryflow/hotfix/<fix_name>/env/<env_name>` branch from `env/<env_name>` branch
    
    - Cherry-pick all commits from `hotfix/<fix_name>` to `cherryflow/hotfix/<fix_name>/env/<env_name>` branch
    
    - Create PR

```yaml
  Title:              "Draft: Merge 'cherryflow/feature/<fix_name>/env/<env_name>' into env/<env_name>",
  Description:        "TODO",
  SourceBranch:       "cherryflow/feature/<fix_name>/env/<env_name>",
  TargetBranch:       "env/<env_name>",
  AssigneeID:         "pusher id",
  RemoveSourceBranch: true,
```  

#### On merged to `env/<env_name>`

- **[RFC/TODO]** Send noti with commits list
