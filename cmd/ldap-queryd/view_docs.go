package main

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/gobuffalo/packr"
	"github.com/sirupsen/logrus"
	"gitlab.com/golang-commonmark/markdown"
)

//TODO write docs
func viewDocs(logger *logrus.Logger) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		box := packr.NewBox("../../../docs")

		var files []string

		box.Walk(func(p string, f packr.File) error {
			files = append(files, p)

			return nil
		})

		sort.Strings(files)

		var content string

		for _, file := range files {
			s, err := box.FindString(file)
			if err != nil {
				//TODO handle error
				//logmsg.With("package", "docs", "function", "View", "msg", "error retrieving doc file contents", "file", file, "error", err.Error())
				//logger.Error(logmsg.Convert())

				//os.Exit(1)
			}

			content = fmt.Sprintf("%s %s", content, s)
		}

		md := markdown.New(markdown.XHTMLOutput(true))

		//TODO HTML header and styling
		//TODO Add content
		//TODO HTML footer
		//TODO Inject content into template?
		//TODO Convert to byte array
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, md.RenderToString([]byte(content)))
	})
}
