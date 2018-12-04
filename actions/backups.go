package actions

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/gobuffalo/buffalo/render"

	"github.com/BakeRolls/gotumblr"
	"github.com/gobuffalo/buffalo"
	"github.com/pkg/errors"
)

// BackupsResource is the resource for the Backup model
type BackupsResource struct {
	buffalo.Resource
}

// Create adds a Backup to the DB. This function is mapped to the
// path POST /blogs/{blog_id}/backups
func (v BackupsResource) Create(c buffalo.Context) error {
	// TODO: Check if blog_id is owned by u
	// u, ok := c.Value("user").(gotumblr.UserInfo)
	// if !ok {
	// 	return errors.New("user not found")
	// }
	t, ok := c.Value("tumblr").(*gotumblr.Client)
	if !ok {
		return errors.New("tumblr client not found")
	}

	w := zip.NewWriter(c.Response())
	defer w.Close()

	var total int
	before := time.Now().Unix()
	last := time.Now()
	for total < 500 {
		c.Logger().Printf("total=%d duration=%v\n", total, time.Now().Sub(last))
		last = time.Now()
		res, err := t.Posts(c.Param("blog_id"), "", url.Values{
			"before": []string{strconv.FormatInt(before, 10)},
		})
		if err != nil {
			return errors.Wrap(err, "could not get posts")
		}
		total += len(res.Posts)
		for _, raw := range res.Posts {
			var p gotumblr.BasePost
			if err := json.Unmarshal(raw, &p); err != nil {
				return errors.Wrap(err, "could not unmarshal base post")
			}
			if err := v.writePost(w, &p, raw); err != nil {
				return errors.Wrap(err, "could not write to zip")
			}
			before = p.Timestamp
		}
		if len(res.Posts) <= 1 {
			break
		}
	}

	if err := w.Close(); err != nil {
		return errors.Wrap(err, "could not close zip")
	}
	return nil
}

func (v BackupsResource) writePost(w /*http.ResponseWriter*/ *zip.Writer, b *gotumblr.BasePost, raw json.RawMessage) error {
	var p interface{} = &gotumblr.BasePost{}
	switch b.PostType {
	case "photo":
		p = &gotumblr.PhotoPost{}
	case "text":
		p = &gotumblr.TextPost{}
	}
	if err := json.Unmarshal(raw, p); err != nil {
		return errors.Wrapf(err, "could not unmarshal %s post", b.PostType)
	}
	switch b.PostType {
	case "photo":
		p := p.(*gotumblr.PhotoPost)
		for _, photo := range p.Photos {
			if err := v.writePhoto(w, photo.OriginalSize.URL); err != nil {
				return errors.Wrapf(err, "could not write photo post %v", b.ID)
			}
		}
	}
	f, err := w.Create(b.ID.String() + ".html")
	if err != nil {
		return errors.Wrapf(err, "could not create post html for %v", b.ID)
	}
	renderer := r.Plain("export/"+b.PostType+".html", "export.html")
	data := render.Data{"post": p, "response": string(raw)}
	if err := renderer.Render(f, data); err != nil {
		return errors.Wrapf(err, "could not render html for %v", b.ID)
	}
	return nil
}

func (v BackupsResource) writePhoto(w *zip.Writer, url string) error {
	f, err := w.Create(path.Join("photos", path.Base(url)))
	if err != nil {
		return errors.Wrap(err, "could not create photo in zip")
	}
	res, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "could not get photo")
	}
	defer res.Body.Close()
	if _, err := io.Copy(f, res.Body); err != nil {
		return errors.Wrap(err, "could not copy photo")
	}
	return nil
}
