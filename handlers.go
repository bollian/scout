package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"scout/data"
	"strconv"
	"strings"
	"time"
)

type safeHandler func(year int, writer http.ResponseWriter, request *http.Request) error

func (handler safeHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	year := time.Now().Year()

	if net.ParseIP(request.Host) != nil {
		handleError(handler(year, writer, request), writer)
	}

	hostParts := strings.Split(request.Host, ".")
	if len(hostParts) == 2 {
		tmp, err := strconv.ParseInt(hostParts[0], 10, 32)
		year = int(tmp)
		if err != nil || year < 2017 || year > time.Now().Year() {
			handleError(data.ErrNotFound, writer)
			return
		}
	} else if len(hostParts) > 2 {
		handleError(data.ErrNotFound, writer)
		return
	}

	handleError(handler(year, writer, request), writer)
}

func handleError(err error, writer http.ResponseWriter) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "handleError: "+err.Error())

	code := http.StatusInternalServerError // the default error value
	switch err {
	case data.ErrNotFound:
		code = http.StatusNotFound
	case data.ErrHTTPMethodUnsupported:
		code = http.StatusMethodNotAllowed
	case data.ErrAccessDenied:
		code = http.StatusUnauthorized
	}

	writer.WriteHeader(code)
	writer.Write([]byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
	<head>
		<title>Error %d</title>
	</head>
	<body>
		Sorry, an error occured (%d).
	</body>
</html>`, code, code)))
}

func indexHandler(year int, writer http.ResponseWriter, request *http.Request) error {
	if request.URL.Path != "/" {
		if page, ok := staticPages[request.URL.Path]; ok {
			return page.ServeHTTP(writer, request)
		}
		return data.ErrNotFound
	}

	return writeAll(writer,
		genPageStart("Scouting System"),
		genStylesheetElement("main"),
		genStylesheetElement("index"),
		genTopBar(request),
		`<h1 class="front-page">main page</h1>`,
		genPageEnd())
}

func loginHandler(year int, writer http.ResponseWriter, request *http.Request) error {
	db, err := data.ConnectToDatabase(year)
	if err != nil {
		return err
	}
	defer db.Close()

	if user := db.GetUser(request); user != nil {
		http.Redirect(writer, request, "/", http.StatusFound)
		return nil
	}

	if request.ParseForm() != nil {
		return data.ErrMalformedRequest
	}

	errorText := ""
	if request.Method == "POST" {
		username := request.PostFormValue("username")
		password := request.PostFormValue("password")
		// don't use the data.Valid* functions here becuase their rules might
		// change over time and someone might just be providing an old
		// username/password.
		if username == "" || password == "" {
			errorText = "Invalid username or password."
			// then continue on to normal page delivery
		} else {
			userCookie, err := db.Login(username, password)
			if err == nil {
				http.SetCookie(writer, userCookie)
				http.Redirect(writer, request, "/", http.StatusFound)
				return nil
			}

			if err == data.ErrUsernameNotFound {
				errorText = "Username not found"
			} else if err == data.ErrPasswordMismatch {
				errorText = "Password was incorrect"
			} else {
				return err
			}
		}
	} else if request.Method != "GET" {
		return data.ErrHTTPMethodUnsupported
	}

	// if a username was provided to the get request, automatically fill in the username textbox
	usernameFiller := request.FormValue("username")
	return writeAll(writer,
		genPageStart("Login"),
		genStylesheetElement("main"),
		genStylesheetElement("login"),
		fmt.Sprintf(`
<div class="login-container">
	<span class="login-error">%s</span>
	<form name="login" action="login" method="post" accept-charset="utf-8">
		<label>Login</label>
		%s
		%s
		<input type="submit" value="Login">
	</form>
</div>`, errorText, genTextInput("username", usernameFiller, "Username", true), genPasswordForm("password", "Password")),
		genPageEnd())
}

func signupHandler(year int, writer http.ResponseWriter, request *http.Request) error {
	db, err := data.ConnectToDatabase(year)
	if err != nil {
		return err
	}
	defer db.Close()

	if user := db.GetUser(request); user != nil {
		http.Redirect(writer, request, "/", http.StatusFound)
		return nil
	}

	if request.ParseForm() != nil {
		return data.ErrMalformedRequest
	}

	username := ""
	realname := ""
	var team int64 = -1
	if request.Method == "POST" {
		username = request.PostFormValue("username")
		realname = request.PostFormValue("realname")
		password := request.PostFormValue("password")
		passwordConf := request.PostFormValue("passwordconf")
		team, err = strconv.ParseInt(request.PostFormValue("team-number"), 10, 16)
		if err != nil || team < 0 {
			deliverSignup(writer, "The team number was not valid (make sure it's greater than 0)", username, realname, -1)
			return nil
		}
		if !data.ValidUsername(username) {
			deliverSignup(writer, "The username was invalid", username, realname, -1)
			return nil
		}
		if !data.ValidRealName(realname) {
			deliverSignup(writer, "The real name provided was invalid", username, realname, -1)
			return nil
		}
		if !data.ValidPassword(password) {
			deliverSignup(writer, "The password was invalid", username, realname, -1)
			return nil
		}
		if password != passwordConf {
			deliverSignup(writer, "The password and the password confirmation didn't match", username, realname, -1)
			return nil
		}

		adminUsername := request.PostFormValue("admin-username")
		adminPassword := request.PostFormValue("admin-password")
		// don't use the data.Valid* functions here becuase their rules might
		// change over time and someone might just be providing an old
		// username/password.
		if adminUsername == "" || adminPassword == "" {
			deliverSignup(writer, "Admin credentials were invalid", username, realname, int(team))
			return nil
		}

		cookie, err := db.CreateUser(username, realname, password, int(team), adminUsername, adminPassword)
		if err == nil {
			http.SetCookie(writer, cookie)
			http.Redirect(writer, request, "/", http.StatusFound)
			return nil
		}

		if err == data.ErrUsernameNotFound { // referring to the admin's username
			deliverSignup(writer, "That admin wasn't found", username, realname, int(team))
			return nil
		} else if err == data.ErrAdminPasswordMismatch {
			deliverSignup(writer, "The admin's password was incorrect", username, realname, int(team))
			return nil
		} else if err == data.ErrUsernameTaken {
			deliverSignup(writer, "Sorry, that username is taken", username, realname, int(team))
			return nil
		}
		return err
	} else if request.Method == "GET" {
		username = request.FormValue("username")
		realname = request.FormValue("realname")
		team, err = strconv.ParseInt(request.FormValue("team-number"), 10, 16)
		if err != nil {
			team = -1
		}
		deliverSignup(writer, "", username, realname, int(team))
	} else {
		return data.ErrHTTPMethodUnsupported
	}
	return nil
}

func deliverSignup(writer http.ResponseWriter, errorText, username, realname string, team int) error {
	page := fmt.Sprintf(`
<span class="signup-error">%s</span>
<form name="create-account" action="signup" method="post" accept-charset="utf-8">
	<div class="new-account">
		<label>New Account</label>
		%s
		%s
		%s
		%s
		%s
	</div>
	<div class="admin-account">
		<label>Admin Confirmation</label>
		%s
		%s
	</div>
	<div class="signup-button-container">
		<input type="submit" value="Create Account"><span class="account-info">Accounts must be recreated at the beginning of each year</span>
	</div>
</form>`, errorText,
		genTextInput("username", username, "Username", true),
		genTextInput("realname", realname, "Full Name", true),
		genTeamNumberForm(team),
		genPasswordForm("password", "Password"),
		genPasswordForm("passwordconf", "Password Confirmation"),
		genTextInput("admin-username", "", "Admin Username", true),
		genPasswordForm("admin-password", "Admin Password"))

	return writeAll(writer,
		genPageStart("Signup"),
		page,
		genPageEnd())
}

func logoutHandler(year int, writer http.ResponseWriter, request *http.Request) error {
	db, err := data.ConnectToDatabase(year)
	if err != nil {
		return err
	}
	defer db.Close()

	db.Logout(writer, request)
	http.Redirect(writer, request, "/", http.StatusFound)
	return nil
}

// writeAll simply accumulates all the errors of multiple writer.Write function calls
func writeAll(writer io.Writer, contents ...string) error {
	var err error
	for _, s := range contents {
		_, err = writer.Write([]byte(s))
		if err != nil {
			return err
		}
	}
	return nil
}
