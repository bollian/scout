package server

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"scout/data"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	authCookieName = "USER"
	bcryptCost     = bcrypt.DefaultCost + 1
	maxAuthId      = math.MaxInt64
)

// Login checks a username and password and returns a secure cookie for future
// authentication if the user is considered valid.  If not valid, an error is
// returned indicating the problem.
func (server *Server) Login(username string, password string) (*http.Cookie, error) {
	var passhash []byte
	safeUsername := EscapeSQLString(username) // make the username string injection-free

	row := server.credStore.QueryRow(fmt.Sprintf("SELECT passhash FROM users WHERE username='%s'", safeUsername))
	err := row.Scan(&passhash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, data.ErrUsernameNotFound
		}
		return nil, errors.New("scout/server: unknown users database error")
	}
	if bcrypt.CompareHashAndPassword(passhash, []byte(password)) != nil {
		return nil, data.ErrPasswordMismatch
	}

	authid, err := genAuthId()
	if err != nil {
		return nil, err
	}

	// update the authid
	_, err = server.credStore.Exec(fmt.Sprintf("UPDATE users SET authid=%d WHERE username='%s'", authid, username))
	if err != nil {
		return nil, data.ErrDatabaseUpdate
	}

	return createAuthCookie(username, authid), nil
}

// Logout attempts to logout the user specified in the http request.  If the
// request doesn't contain the valid identity of a user, or the user doesn't
// exist, Logout returns false.
func (server *Server) Logout(request *http.Request) bool {
	user := server.GetUser(request)
	if user == nil {
		return false
	}
	query := fmt.Sprintf("UPDATE users SET authid=-1 WHERE username='%s' && authid=%d", user.Username, user.Id)
	_, err := server.credStore.Exec(query)
	if err != nil {
		return false
	}
	return true
}

// GetUser determines if an http request is coming from a client that is currently
// logged in.  If the client is logged in, a struct containing the user info is
// returned, else nil.
func (server *Server) GetUser(request *http.Request) *data.User {
	if request == nil {
		return nil
	}

	cookie := GetAuthCookie(request.Cookies())
	if cookie == nil {
		return nil
	}

	username, authid := parseAuthCookie(cookie)
	if username == "" || authid == -1 {
		return nil
	}

	var (
		id       int64
		realname string
	)
	now := data.Now()
	row := server.credStore.QueryRow(fmt.Sprintf(
		`SELECT id, username, realname
 FROM users
 WHERE username='%s' && authid=%d && lastseen BETWEEN '%s' AND NOW()`, username, authid, now.Add(time.Minute*-30)))
	err := row.Scan(&id, &username, &realname)
	if err != nil {
		return nil
	}

	return &data.User{
		Username: username,
		RealName: realname,
		Id:       id,
		GameYear: now.Year(),
	}
}

// CreateUser adds a user to the credential store.  Returns the authentication
// cookie, so no login is required after successfully creating a new user.  If
// the user could not be created, an error is returned.
func (server *Server) CreateUser(username, realname, password string, team int) (*http.Cookie, error) {
	if !ValidUsername(username) {
		return nil, data.InvalidUsername{Username: username}
	}
	safeUsername := EscapeSQLString(username)

	passhash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}

	authid, err := genAuthId()
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`INSERT INTO users (username, realname, bcrypt, authid) VALUE ('%s', '%s', '%s', %d)`,
		safeUsername, realname, string(passhash), authid)
	_, err = server.credStore.Exec(query)
	if err != nil {
		return nil, data.UsernameTakenError{Username: username}
	}

	return createAuthCookie(username, authid), nil
}

// EscapeSQLString takes a string that would open a vulnerability to sql
// injections and applies the proper escape sequences to make it safe for use.
// Do not include leading and trailing quotes in the string.
func EscapeSQLString(s string) string {
	s = strings.Replace(s, "\\", "\\\\", -1)
	s = strings.Replace(s, "'", "\\'", -1)
	s = strings.Replace(s, "\"", "\\\"", -1)
	return s
}

func genAuthId() (*big.Int, error) {
	authid, err := rand.Int(rand.Reader, big.NewInt(maxAuthId))
	if err != nil {
		return nil, data.ErrRandGeneration
	}
	return authid, nil
}

func createAuthCookie(username string, authid *big.Int) *http.Cookie {
	return &http.Cookie{
		Name:   authCookieName,
		Secure: true,
		Value:  username + "$" + authid.String(),
		MaxAge: 7200, // 2 hours in seconds
	}
}

// parseAuthCookie determines the username and the authid of the user
// represented by the cookie.  If there is no valid authid, -1 is returned.  If
// there is no valid username, an empty string is returned.
func parseAuthCookie(cookie *http.Cookie) (string, int64) {
	delimeter := strings.LastIndex(cookie.Value, "$")
	if delimeter == -1 {
		return "", -1
	}

	id, err := strconv.ParseInt(cookie.Value[delimeter+1:], 10, 64)
	if err != nil {
		id = -1
	}

	username := cookie.Value[:delimeter]
	if !ValidUsername(username) {
		username = ""
	}

	return username, id
}

// ValidUsername checks to make sure that username is an acceptable username
func ValidUsername(username string) bool {
	return username != "" && !strings.ContainsAny(username, `!@#$%^&*~+'"`)
}

// GetAuthCookie finds the cookie from the slice that identifies a user.
// Returns nil if none is found.
func GetAuthCookie(cookies []*http.Cookie) *http.Cookie {
	for _, c := range cookies {
		if c.Name == authCookieName {
			return c
		}
	}
	return nil
}
