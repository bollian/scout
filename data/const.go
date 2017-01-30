package data

import "errors"

const (
	// MinYear is the first year the scouting system was active
	MinYear = 2017
)

var (
	// ErrWrongYear is returned whenever data is requested for a game year during which the
	// scouting system hasn't been active.
	ErrWrongYear = errors.New("the scouting system was not in operation for the provided year")

	// ErrAccessDenied indicates that a resource was requested that required
	// privileges the client didn't have.  Normally used for simple http
	// authentication.
	ErrAccessDenied = errors.New("access denied")
	// ErrInvalidUsername indicates that a provided username doesn't meet the required standards
	// for creating new usernames
	ErrInvalidUsername = errors.New("invalid username")
	// ErrUsernameTaken indicates that the username requested for a new account is already in use
	ErrUsernameTaken = errors.New("username taken")
	// ErrUsernameNotFound indicates that a user wasn't found in the credentials table
	ErrUsernameNotFound = errors.New("username not found")
	// ErrPasswordMismatch indicates that a provided password didn't match what was stored for the user
	ErrPasswordMismatch = errors.New("password mismatch")
	// ErrAdminPasswordMismatch is the same as ErrPasswordMismatch except it applies only to passwords that
	// were provided by an admin (such as when creating a user)
	ErrAdminPasswordMismatch = errors.New("admin password mismatch")
	// ErrDatabaseUpdate indicates that an update operation failed
	ErrDatabaseUpdate = errors.New("could not update database")
	// ErrRandGeneration indicates that it was impossible to generate a cryptorandom number
	ErrRandGeneration = errors.New("could not create a strong random number")

	// ErrNotFound is an HTTP page not found error
	ErrNotFound = errors.New("page not found")
	// ErrHTTPMethodUnsupported indicates that a page was requested using an inapropriate HTTP method
	ErrHTTPMethodUnsupported = errors.New("http method not supported")
	// ErrMalformedRequest indicates that there was something wrong with the contents of an HTTP
	// request that made it impossible to understand.
	ErrMalformedRequest = errors.New("http request wasn't understandable")
)
