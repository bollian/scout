package main

import (
	"fmt"
	"net/http"
	"os"
)

const (
	exitSuccess = iota
	exitCacheError
	exitServeError
)

func main() {
	var exitCode int = program()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func program() int {
	err := cacheStaticPages()
	if err != nil {
		fmt.Fprintln(os.Stderr, "caching error: "+err.Error())
		return exitCacheError
	}
	setupHandlers()

	err = http.ListenAndServe(":80", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hosting error: "+err.Error())
		return exitServeError
	}
	return 0
}
