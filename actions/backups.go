package actions

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/BakeRolls/gotumblr"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/pkg/errors"
)

// BackupsResource is the resource for the Backup model
type BackupsResource struct {
	buffalo.Resource
}

// Create adds a Backup to the DB. This function is mapped to the
// path POST /blogs/{blog_id}/backups
func (v BackupsResource) Create(c buffalo.Context) error {
	u, ok := c.Value("user").(gotumblr.UserInfo)
	if !ok {
		return errors.New("user not found")
	}
	var ownBlog bool
	for _, b := range u.Blogs {
		if b.Name == c.Param("blog_id") {
			ownBlog = true
			break
		}
	}
	if !ownBlog {
		return errors.Errorf("blog %s does not belong to %s", c.Param("blog_id"), u.Name)
	}

	t, ok := c.Value("tumblr").(*gotumblr.Client)
	if !ok {
		return errors.New("tumblr client not found")
	}

	h := c.Response().Header()
	h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%d.zip", c.Param("blog_id"), time.Now().Unix()))
	h.Set("Content-Type", mime.TypeByExtension(".zip"))

	w := zip.NewWriter(c.Response())
	defer w.Close()

	includeReblogs := c.Param("include_reblogs") == "true"
	before := time.Now().Unix()
	for {
		res, err := t.Posts(c.Param("blog_id"), "", url.Values{
			"before":      []string{strconv.FormatInt(before, 10)},
			"reblog_info": []string{"true"},
		})
		if err != nil {
			return errors.Wrap(err, "could not get posts")
		}
		for _, raw := range res.Posts {
			var p gotumblr.BasePost
			if err := json.Unmarshal(raw, &p); err != nil {
				return errors.Wrap(err, "could not unmarshal base post")
			}
			before = p.Timestamp
			if p.RebloggedFromID != "" && !includeReblogs {
				continue
			}
			if err := v.writePost(w, &p, raw); err != nil {
				return errors.Wrap(err, "could not write to zip")
			}
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

func (v BackupsResource) writePost(w *zip.Writer, b *gotumblr.BasePost, raw json.RawMessage) error {
	var p interface{} = &gotumblr.BasePost{}
	switch b.PostType {
	case "answer":
		p = &gotumblr.AnswerPost{}
	case "chat":
		p = &gotumblr.ChatPost{}
	case "photo":
		p = &gotumblr.PhotoPost{}
	case "text":
		p = &gotumblr.TextPost{}
	}
	if err := json.Unmarshal(raw, p); err != nil {
		return errors.Wrapf(err, "could not unmarshal %s post", b.PostType)
	}
	if err := v.writeRaw(w, b.ID.String(), raw); err != nil {
		return errors.Wrap(err, "could not write raw API response")
	}
	data := render.Data{"post": p, "response": string(raw)}
	switch b.PostType {
	case "photo":
		p := p.(*gotumblr.PhotoPost)
		if err := v.writePhoto(w, p, data); err != nil {
			return errors.Wrap(err, "could not write photo")
		}
	}
	f, err := w.Create(b.ID.String() + ".html")
	if err != nil {
		return errors.Wrapf(err, "could not create post html for %v", b.ID)
	}
	renderer := r.Plain("export/"+b.PostType+".html", "export.html")
	if err := renderer.Render(f, data); err != nil {
		return errors.Wrapf(err, "could not render html for %v", b.ID)
	}
	return nil
}

func (v BackupsResource) writeRaw(w *zip.Writer, id string, raw json.RawMessage) error {
	f, err := w.Create(path.Join("raw", id+".json"))
	if err != nil {
		return errors.Wrap(err, "could not create raw API response in zip")
	}
	if _, err := f.Write(raw); err != nil {
		return errors.Wrap(err, "could not write raw API response to zip")
	}
	return nil
}

func (v BackupsResource) writePhoto(w *zip.Writer, p *gotumblr.PhotoPost, data render.Data) error {
	var photos []string
	for _, photo := range p.Photos {
		url := photo.OriginalSize.URL
		res, err := http.Get(url)
		if err != nil {
			return errors.Wrap(err, "could not get photo")
		}
		defer res.Body.Close()
		f, err := w.Create(path.Join("photos", path.Base(url)))
		if err != nil {
			return errors.Wrap(err, "could not create photo in zip")
		}
		if _, err := io.Copy(f, res.Body); err != nil {
			return errors.Wrap(err, "could not write photo")
		}
		res.Body.Close()
	}
	data["photos"] = photos
	return nil
}
