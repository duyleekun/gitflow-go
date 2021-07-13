module github.com/duyleekun/gitflow-go/webhook

go 1.16

require (
	github.com/duyleekun/gitflow-go/shared v0.0.0
	github.com/go-playground/webhooks/v6 v6.0.0-beta.3
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/xanzy/go-gitlab v0.50.1
)

replace github.com/duyleekun/gitflow-go/shared v0.0.0 => ../shared
