module github.com/duyleekun/gitflow-go/webhook

go 1.16

require (
	github.com/go-playground/webhooks/v6 v6.0.0-beta.3
	github.com/xanzy/go-gitlab v0.50.1
	github.com/duyleekun/gitflow-go/shared v0.0.0
)

replace (
	"github.com/duyleekun/gitflow-go/shared" v0.0.0 => "../shared"
)