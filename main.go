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
			if !server.Cache("/"+info.Name(), data, contentType) {
				return errors.New("duplicate cache names")
			}
			return nil
		})
		if err != nil {
			log.Printf("Error constructing cache (%s)\n", err.Error())
			return exitCacheError
		}
	}

	server.Handle("/login", server.HandlerFunc(loginHandler))
	server.Handle("/logout", server.HandlerFunc(logoutHandler))
	server.Handle("/signup", server.HandlerFunc(signupHandler))

	server.Handle("/competitions", server.HandlerFunc(competitionsHandler))
	server.Handle("/teams", server.HandlerFunc(teamsHandler))
	server.Handle("/all", server.HandlerFunc(allDataHandler))
	server.Handle("/matches", server.HandlerFunc(matchesHandler))
	server.Handle("/submit", server.HandlerFunc(submissionHandler))

	server.Handle("/", server.HandlerFunc(indexHandler))

	err := server.ListenAndServe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running listener: %s\n", err.Error())
		return exitListenerError
	}

	return exitSuccess
}
