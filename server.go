package main

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

var (
	staticPages map[string]staticPage = map[string]staticPage{}

	errDuplicateCache       = errors.New("duplicate cache names")
	errUnknownFileExtension = errors.New("file type named by file extension not supported")
)

type staticPage struct {
	contents    []byte
	contentType string
}

func (page staticPage) ServeHTTP(writer http.ResponseWriter, request *http.Request) error {
	writer.Header().Set("Content-Type", page.contentType)
	_, err := writer.Write(page.contents)
	return err
}

func cacheStaticPages() error {
	directories := []string{
		"css",
		"js",
		"img",
	}
	for _, dir := range directories {
		err := filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
			if info == nil || info.IsDir() {
				return nil // ignore directories and the root of the walk
			}
			if _, ok := staticPages["/"+info.Name()]; ok {
				return errDuplicateCache
			}

			f, ferr := os.Open(name)
			if ferr != nil {
				return ferr
			}
			defer f.Close()

			buffer := bytes.NewBuffer(make([]byte, 0, info.Size()))
			_, ferr = buffer.ReadFrom(f)
			if ferr != nil {
				return ferr
			}

			contentType := ""
			switch filepath.Ext(info.Name()) {
			case ".js":
				contentType = "text/javascript"
			case ".css":
				contentType = "text/css"
			case ".jpg", ".jpeg", ".ico":
				contentType = "image/jpeg"
			default:
				return errUnknownFileExtension
			}

			staticPages["/"+info.Name()] = staticPage{contents: buffer.Bytes(), contentType: contentType}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func setupHandlers() {
	http.Handle("/login", safeHandler(loginHandler))   // login as a user
	http.Handle("/logout", safeHandler(logoutHandler)) // logout and redirect to main page
	http.Handle("/signup", safeHandler(signupHandler)) // signup w/ admin authorization

	// http.Handle("/competitions", nil) // a list of all competitions and all the teams that attend them
	// http.Handle("/teams", nil)        // a list of all teams w/ their track records
	// http.Handle("/matches", nil)      // a list of all matches w/ general scorint info
	// http.Handle("/all", nil)          // a list of all submissions w/ brief overviews
	// http.Handle("/detailed", nil)     // a view of single submissions in full detail
	// http.Handle("/analysis", nil)     // a view of robots ranked for certain characteristics

	// http.Handle("/submit", nil) // submit a new entry into the data collection

	http.Handle("/", safeHandler(indexHandler)) // the main page; also handles static pages for resource files
}
