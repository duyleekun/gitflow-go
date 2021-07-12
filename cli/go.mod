module github.com/duyleekun/gitflow-go/cli

go 1.16

require (
	github.com/duyleekun/gitflow-go/shared v0.0.0
	github.com/manifoldco/promptui v0.8.0
	github.com/xanzy/go-gitlab v0.50.1
)

replace github.com/duyleekun/gitflow-go/shared v0.0.0 => ../shared
