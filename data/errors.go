package data

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var (
	// ErrAccess indicates that a resource was requested that required
	// privileges the client didn't have.  Normally used for simple http
	// authentication.
	ErrAccess = errors.New("insufficient access privileges")
	// ErrUsernameNotFound indicates that the requested user is not contained
	// in the accounts list
	ErrUsernameNotFound = errors.New("user not found")
	// ErrPasswordMismatch indicates that the password provided for a user didn't
	// match what was stored in the database
	ErrPasswordMismatch = errors.New("password mismatch")
	// ErrDatabaseUpdate indicates that an update operation failed
	ErrDatabaseUpdate = errors.New("could not update database")
	// ErrRandGeneration indicates that it was impossible to generate a cryptorandom number
	ErrRandGeneration = errors.New("could not create a strong random number")
)

// HTTPMethodError indicates that an inapropriate http method was used for the
// request (GET, POST, etc.)
type HTTPMethodError struct {
	Method string
}

func (err HTTPMethodError) Error() string {
	return fmt.Sprintf("method %s not allowed", err.Method)
}

// UsernameTakenError is returned when CreateUser finds a username is already in
// the users database
type UsernameTakenError struct {
	Username string
}

func (err UsernameTakenError) Error() string {
	return fmt.Sprintf("username '%s' already taken", err.Username)
}

// InvalidUsername indicates that a username was either empty or contained
// disallowed characters
type InvalidUsername struct {
	Username string
}

func (err InvalidUsername) Error() string {
	return fmt.Sprintf("username '%s' is invalid", err.Username)
}

// HTTPRequestParseError is used to represent an error occuring from a malformed
// http request, causing a bad parse
type HTTPRequestParseError struct {
	Request *http.Request
}

func (err HTTPRequestParseError) Error() string {
	msg := fmt.Sprintf("unable to parse http request to %s", err.Request.URL.String())
	if err.Request.ContentLength > -1 {
		content := make([]byte, err.Request.ContentLength)
		_, newErr := err.Request.Body.Read(content)
		if newErr == nil {
			msg += fmt.Sprintf(" containing '%s'", string(content))
		}
	}
	return msg
}

// HTTPNotFoundError means the requested page was not found on the server
type HTTPNotFoundError struct {
	URL *url.URL
}

func (err HTTPNotFoundError) Error() string {
	return fmt.Sprintf("page not found: %s", err.URL.String())
}
