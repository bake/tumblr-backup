package actions

import (
	"net/http"

	"github.com/bake/gotumblr"
	"github.com/gobuffalo/buffalo"
	"github.com/pkg/errors"
)

// BlogsResource is the resource for the Blog model
type BlogsResource struct {
	buffalo.Resource
}

// List gets all Blogs. This function is mapped to the path
// GET /blogs
func (v BlogsResource) List(c buffalo.Context) error {
	u, ok := c.Value("user").(gotumblr.UserInfo)
	if !ok {
		return errors.New("tumblr client not found")
	}
	c.Set("blogs", u.Blogs)
	return c.Render(http.StatusOK, r.HTML("blogs/list.html"))
}

// Show gets the data for one Blog. This function is mapped to
// the path GET /blogs/{blog_id}
func (v BlogsResource) Show(c buffalo.Context) error {
	t, ok := c.Value("tumblr").(*gotumblr.Client)
	if !ok {
		return errors.WithStack(errors.New("no tumblr client found"))
	}

	res, err := t.BlogInfo(c.Param("blog_id"))
	if err != nil {
		return errors.Wrap(err, "could not get blog")
	}
	c.Set("blog", res.Blog)

	return c.Render(http.StatusOK, r.HTML("blogs/show.html"))
}
