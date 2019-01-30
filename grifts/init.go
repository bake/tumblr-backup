package grifts

import (
	"git.192k.pw/tumblr/backup/actions"
	"github.com/gobuffalo/buffalo"
)

func init() {
	buffalo.Grifts(actions.App())
}
