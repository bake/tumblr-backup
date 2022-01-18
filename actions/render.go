package actions

import (
	"path"

	"git.192k.pw/tumblr/backup/public"
	"git.192k.pw/tumblr/backup/templates"
	"github.com/gobuffalo/buffalo/render"
)

var r *render.Engine

func init() {
	r = render.New(render.Options{
		// HTML layout to be used for all HTML requests:
		HTMLLayout: "application.plush.html",

		// Box containing all of the templates:
		TemplatesFS: templates.FS(),
		AssetsFS:    public.FS(),

		// Add template helpers here:
		Helpers: render.Helpers{
			"base": path.Base,
			"even": func(i int) bool { return i%2 == 0 },
		},
	})
}
