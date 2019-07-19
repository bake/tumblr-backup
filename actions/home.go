package actions

import (
	"net/http"

	"github.com/bake/gotumblr"
	"github.com/gobuffalo/buffalo"
)

// HomeHandler is a default handler to serve up
// a home page.
func HomeHandler(c buffalo.Context) error {
	if _, ok := c.Value("user").(gotumblr.UserInfo); ok {
		return c.Redirect(http.StatusTemporaryRedirect, "/blogs")
	}
	return c.Render(http.StatusOK, r.HTML("index.html"))
}
