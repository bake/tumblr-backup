package grifts

import (
	"git.192k.pw/bake/tumblrbackup/actions"
	"github.com/gobuffalo/buffalo"
)

func init() {
	buffalo.Grifts(actions.App())
}
