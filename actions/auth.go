package actions

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/bake/gotumblr"
	"github.com/bake/httpcache"
	"github.com/bake/httpcache/diskcache"
	"github.com/gobuffalo/buffalo"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/tumblr"
	"github.com/pkg/errors"
)

func init() {
	gothic.Store = App().SessionStore

	goth.UseProviders(tumblr.New(
		os.Getenv("TUMBLR_KEY"),
		os.Getenv("TUMBLR_SECRET"),
		fmt.Sprintf("%s%s", app.Options.Host, "/auth/tumblr/callback"),
	))
}

func AuthHandler(c buffalo.Context) error {
	user, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		return c.Error(http.StatusUnauthorized, err)
	}
	c.Session().Set("current_user", user.Name)
	c.Session().Set("access_token", user.AccessToken)
	c.Session().Set("access_token_secret", user.AccessTokenSecret)
	if err := c.Session().Save(); err != nil {
		return errors.WithStack(err)
	}
	c.Flash().Add("success", fmt.Sprintf("Welcome %s!", user.Name))
	return c.Redirect(http.StatusTemporaryRedirect, "/blogs")
}

func LogoutHandler(c buffalo.Context) error {
	c.Session().Clear()
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func SetCurrentUser(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		token, ok := c.Session().Get("access_token").(string)
		if !ok {
			c.Session().Clear()
			return next(c)
		}
		secret, ok := c.Session().Get("access_token_secret").(string)
		if !ok {
			c.Session().Clear()
			return next(c)
		}
		userName, ok := c.Session().Get("current_user").(string)
		if !ok {
			c.Session().Clear()
			return next(c)
		}
		trans := httpcache.New(
			diskcache.New(path.Join("cache", userName), time.Hour),
			httpcache.WithVerifier(httpcache.StatusInTwoHundreds),
		)
		t := gotumblr.New(
			os.Getenv("TUMBLR_KEY"), os.Getenv("TUMBLR_SECRET"), token, secret,
			gotumblr.SetClient(trans.Client()),
		)
		c.Set("tumblr", t)

		res, err := t.Info()
		if err != nil {
			return errors.Wrap(err, "could not get user info")
		}
		c.Set("user", res.User)
		c.LogField("user", res.User.Name)

		return next(c)
	}
}
