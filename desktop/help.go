package desktop

import (
	"bytes"
	"github.com/shruggietech/insonic/internal/contracts"
	"golang.org/x/net/html"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strings"
)

func verifyHelp() error {
	pages := 0
	err := fs.WalkDir(Assets, "assets/help", func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(file, ".html") {
			return nil
		}
		pages++
		data, err := Assets.ReadFile(file)
		if err != nil {
			return err
		}
		tokens := html.NewTokenizer(bytes.NewReader(data))
		for {
			switch tokens.Next() {
			case html.ErrorToken:
				if tokens.Err() == io.EOF {
					return nil
				}
				return tokens.Err()
			case html.StartTagToken, html.SelfClosingTagToken:
				for _, attribute := range tokens.Token().Attr {
					if attribute.Key != "src" && attribute.Key != "href" {
						continue
					}
					link, err := url.Parse(attribute.Val)
					if err != nil {
						return err
					}
					if link.Scheme != "" || link.Host != "" || link.Path == "" {
						continue
					}
					target := path.Clean(path.Join(path.Dir(file), link.Path))
					if strings.HasPrefix(link.Path, "/") {
						target = path.Join("assets/help", link.Path)
					}
					if !strings.HasPrefix(target, "assets/help/") && target != "assets/help" {
						return contracts.Fail("unavailable")
					}
					info, err := fs.Stat(Assets, target)
					if err != nil {
						return err
					}
					if info.IsDir() {
						if _, err := Assets.ReadFile(path.Join(target, "index.html")); err != nil {
							return err
						}
					}
				}
			}
		}
	})
	if err != nil || pages == 0 {
		return contracts.Fail("unavailable")
	}
	return nil
}
