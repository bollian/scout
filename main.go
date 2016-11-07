package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"scout/server"
)

const (
	exitSuccess = iota
	exitNewServerError
	exitCacheError
	exitListenerError
)

func main() {
	exit := program()
	if exit != exitSuccess {
		os.Exit(exit)
	}
}

func program() int {
	website, err := server.New(":80")
	if website == nil {
		log.Printf("Unable to initialize server (%s)\n", err.Error())
		return 1
	}

	folders := map[string]string{
		"css": "text/css",
		"js":  "text/javascript",
		"img": "image/jpeg",
	}
	for folder, contentType := range folders {
		err := filepath.Walk(folder, func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			f, ferr := os.Open(p)
			if ferr != nil {
				return ferr
			}
			defer f.Close()

			data := make([]byte, info.Size())
			f.Read(data)
			if !website.Cache("/"+info.Name(), data, contentType) {
				return errors.New("duplicate cache names")
			}
			return nil
		})
		if err != nil {
			log.Printf("Error constructing cache (%s)\n", err.Error())
			return exitCacheError
		}
	}

	website.Handle("/login", server.HandlerFunc(loginHandler))
	website.Handle("/logout", server.HandlerFunc(logoutHandler))
	website.Handle("/signup", server.HandlerFunc(signupHandler))

	website.Handle("/competitions", server.HandlerFunc(competitionsHandler))
	website.Handle("/teams", server.HandlerFunc(teamsHandler))
	website.Handle("/all", server.HandlerFunc(allDataHandler))
	website.Handle("/matches", server.HandlerFunc(matchesHandler))
	website.Handle("/submit", server.HandlerFunc(submissionHandler))

	website.Handle("/", server.HandlerFunc(indexHandler))

	err = website.ListenAndServe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running listener: %s\n", err.Error())
		return exitListenerError
	}

	return exitSuccess
}
