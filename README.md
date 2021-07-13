# gitflow-go

Automated repetitive setup actions and git actions

<!--ts-->
   * [gitflow-go](#gitflow-go)
      * [CLI for Setup](#cli-for-setup)
         * [Maintainer usage](#maintainer-usage)
      * [Webhook for Automation](#webhook-for-automation)
         * [Developer Usage](#developer-usage)
         * [Webhook supported events](#webhook-supported-events)
            * [On feature/&lt;feature_name&gt; or hotfix/&lt;fix_name&gt; pushed](#on-featurefeature_name-or-hotfixfix_name-pushed)
            * [On feature/&lt;feature_name&gt; merged to main](#on-featurefeature_name-merged-to-main)
            * [On hotfix/&lt;fix_name&gt; merged to main](#on-hotfixfix_name-merged-to-main)
            * [On merged to env/&lt;env_name&gt;](#on-merged-to-envenv_name)

<!-- Added by: root, at: Mon Jul 12 05:33:03 UTC 2021 -->

<!--te-->

## CLI for Setup

Developer can skip this section, this CLI is just for setup auto Gitflow for new project

### Maintainer usage 

0. build & run CLI
   
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

- Enter Webhook URL

- **[DANGER]** Delete all other hooks
- Add new webhook

## Webhook for Automation

### Developer Usage

- To merge `PR`
  + Go to `PR` on Gitlab
  + Press `rebase` to perform FastForward merge without creating merge commit
    + If `rebase` failed when merge from `main` or `env/<envname>` to `env/<env_name>`
      + Comment `!force_merge` 
    + If `rebase` failed on other source and target branches, something is wrong
  + Update `squash` message (for changelog)
  + Press `merge`
  

- New feature 
  + Create branch named `feature/<feature_name>` from `main`
  + When done, merge `PR` from `feature/<feature_name>` to `main`


- Hotfix
  + Create branch named `hotfix/<fix_name>` from `main`
  + When done, merge `PR` from `hotfix/<fix_name>` to `main`
  + Merge `PR` from `cherryflow/hotfix/<fix_name>/env/<env_name>` to `env/<env_name>` 


- Deploy to environments `env/*`
  + Create `PR` from anything to `env/*`
  + Merge `PR` to deploy


### Webhook supported events

- `RFC` means not finalize, need idea
- `TODO` not implemented

### Helm

```shell
kubectl create ns gitflow
helm upgrade --install --namespace=gitflow devns ./helm --values values-override.yaml
```


#### On `feature/<feature_name>` or `hotfix/<fix_name>` pushed

- Create `PR`

```yaml
  Title:              "Draft: Merge 'feature/<feature_name> or hotfix/<fix_name>' into main",
  Description:        "TODO",
  SourceBranch:       "feature/<feature_name> or hotfix/<fix_name>",
  TargetBranch:       "main",
  AssigneeID:         "pusher id",
  RemoveSourceBranch: true,
```

#### On `feature/<feature_name>` merged to `main`
- **[RFC]****[TODO]** auto create `PR` from `main` to ???


#### On `hotfix/<fix_name>` merged to `main`

- For each `env/<env_name>` branch 

    - Auto create `cherryflow/hotfix/<fix_name>/env/<env_name>` branch from `env/<env_name>` branch
    
    - Cherry-pick all commits from `hotfix/<fix_name>` to `cherryflow/hotfix/<fix_name>/env/<env_name>` branch
    
    - Create `PR`

```yaml
  Title:              "Draft: Merge 'cherryflow/feature/<fix_name>/env/<env_name>' into env/<env_name>",
  Description:        "TODO",
  SourceBranch:       "cherryflow/feature/<fix_name>/env/<env_name>",
  TargetBranch:       "env/<env_name>",
  AssigneeID:         "pusher id",
  RemoveSourceBranch: true,
```  

#### On merged to `env/<env_name>`

- Handle `!force_merge` comment
  - Delete target branch
  - Recreate target branch from source branch
  - Close PR
- **[RFC]****[TODO]** Send notification with commits list

## Contributions

### Generate TOC

```shell
docker run -it -v $PWD:/data --rm --name markdown-toc-generator evkalinin/gh-md-toc:0.7.0 --insert /data/README.md
```