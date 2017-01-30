package data

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql" // the init function registers a sql driver
)

const (
	authCookieName = "USER"
	bcryptCost     = bcrypt.DefaultCost + 1
	maxAuthId      = math.MaxInt64
)

// User represents a sinle row from the accounts database
type User struct {
	Id       int64
	Username string
	RealName string
	GameYear int
}

// DB represents a connection to the scouting database
type DB struct {
	db *sql.DB
}

// ConnectToDatabase establishes a connection to the database containing the information for the
// provided year.  Returns a zero-initialized connection in the case of an error.  Remember to
// close the connection after you're done with it.
func ConnectToDatabase(year int) (DB, error) {
	if !VerifyYear(year) {
		return DB{}, ErrWrongYear
	}
	db, err := sql.Open("mysql", "scout@/scouting"+strconv.Itoa(year))
	if err != nil {
		return DB{}, err
	}
	return DB{db: db}, nil
}

// Close frees all resources used by the database connection.
func (db DB) Close() error {
	return db.db.Close()
}

// Login checks a username and password and returns a secure cookie for future
// authentication if the user is considered valid.  If not valid, an error is
// returned indicating the problem.
func (db DB) Login(username string, password string) (*http.Cookie, error) {
	var passhash []byte
	safeUsername := EscapeSQLString(username) // make the username string injection-free

	row := db.db.QueryRow(fmt.Sprintf("SELECT passhash FROM users WHERE username='%s'", safeUsername))
	err := row.Scan(&passhash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUsernameNotFound
		}
		return nil, errors.New("data.Login: unknown users query error")
	}
	if bcrypt.CompareHashAndPassword(passhash, []byte(password)) != nil {
		return nil, ErrPasswordMismatch
	}

	authid, err := genAuthId()
	if err != nil {
		return nil, err
	}

	// update the authid
	_, err = db.db.Exec(fmt.Sprintf("UPDATE users SET authid=%d WHERE username='%s'", authid, username))
	if err != nil {
		return nil, ErrDatabaseUpdate
	}

	return createAuthCookie(username, authid), nil
}

// Logout attempts to logout the user specified in the http request.  If the
// request doesn't contain the valid identity of a user, or the user doesn't
// exist, Logout returns false.
func (db DB) Logout(response http.ResponseWriter, request *http.Request) bool {
	user := db.GetUser(request)
	if user == nil {
		return false
	}

	http.SetCookie(response, createAuthCookie("", big.NewInt(-1)))

	query := fmt.Sprintf("UPDATE users SET authid=-1 WHERE username='%s' && authid=%d", user.Username, user.Id)
	_, err := db.db.Exec(query)
	if err != nil {
		return false // unable to confirm that the user was logged out
	}
	return true
}

// GetUser determines if an http request is coming from a client that is currently
// logged in.  If the client is logged in, a struct containing the user info is
// returned, else nil.
func (db DB) GetUser(request *http.Request) *User {
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
	now := Now()
	row := db.db.QueryRow(fmt.Sprintf(
		`SELECT id, username, realname
 FROM users
 WHERE username='%s' && authid=%d && lastseen BETWEEN '%s' AND NOW()`, username, authid, now.Add(time.Minute*-30)))
	err := row.Scan(&id, &username, &realname)
	if err != nil {
		return nil
	}

	return &User{
		Username: username,
		RealName: realname,
		Id:       id,
		GameYear: now.Year(),
	}
}

// CreateUser adds a user to the credential store.  Returns the authentication
// cookie, so no login is required after successfully creating a new user.  If
// the user could not be created, an error is returned.
func (db DB) CreateUser(username, realname, password string, team int, adminUsername, adminPassword string) (*http.Cookie, error) {
	if !ValidUsername(username) {
		return nil, ErrInvalidUsername
	}

	safeAdminUsername := EscapeSQLString(adminUsername)
	var adminPasshash []byte
	query := fmt.Sprintf("SELECT passhash FROM users WHERE username='%s' && admin=true", safeAdminUsername)
	row := db.db.QueryRow(query)
	if err := row.Scan(&adminPasshash); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUsernameNotFound
		}
		fmt.Fprintln(os.Stderr, "data.CreateUser: "+err.Error())
		return nil, errors.New("data.CreateUser: unknown users query error")
	}
	if bcrypt.CompareHashAndPassword(adminPasshash, []byte(adminPassword)) != nil {
		return nil, ErrAdminPasswordMismatch
	}

	safeUsername := EscapeSQLString(username)
	safeRealname := EscapeSQLString(realname)
	passhash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}

	authid, err := genAuthId()
	if err != nil {
		return nil, err
	}

	query = fmt.Sprintf(`INSERT INTO users (username, realname, passhash, authid) VALUE ('%s', '%s', '%s', %d)`,
		safeUsername, safeRealname, string(passhash), authid)
	_, err = db.db.Exec(query)
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert error: "+err.Error())
		return nil, ErrUsernameTaken
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
		return nil, ErrRandGeneration
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

// ValidPassword checks to see if the given password is an acceptable password.
//
// Currently, the only requirement for passwords is that they be more than 8
// characters in length.
func ValidPassword(password string) bool {
	return len(password) > 8
}

// ValidRealName checks to see if the real, human name was valid.
func ValidRealName(name string) bool {
	return len(name) > 0
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
