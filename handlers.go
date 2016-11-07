package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"scout/data"
	"scout/server"
	"strconv"
	"strings"
)

func indexHandler(writer http.ResponseWriter, request *http.Request) error {
	// first check to make sure we're serving the main page since everything not
	// covered by the other handlers comes here
	if request.URL.Path != "/" {
		return data.HTTPNotFoundError{URL: request.URL}
	}

	startForm("ED Scouting", writer)
	insertStylesheet("index", writer)
	insertTopBar(writer)
	writer.Write([]byte(`<h1 class="front-page">main page</h1>`))
	endForm(writer)

	return nil
}

func competitionsHandler(writer http.ResponseWriter, request *http.Request) error {
	return data.HTTPNotFoundError{URL: request.URL}
}

const (
	frmPassword = `<input name="password" type="password" placeholder="Password" required>`
)

func loginHandler(writer http.ResponseWriter, request *http.Request) error {
	if user := server.GetUser(request); user != nil {
		// if the user is already logged in, redirect to the main page
		http.Redirect(writer, request, "/", http.StatusFound)
		return nil
	}

	var usernameForm string
	err := request.ParseForm()
	if err == nil {
		if request.Method == "POST" {

		} else if request.Method == "GET" {

			usernameForm = genUserForm("")
		}
	}

	startForm("Login", writer)
	insertStylesheet("login", writer)
	writer.Write([]byte(`
<form name="login" action="login" method="post" accept-charset="utf-8">
	<label>Login</label>
	<ul>
		` + usernameForm + `
		` + frmPassword + `
		<li><input type="submit" value="Login"></li>
	</ul>
</form>`))
	endForm(writer)

	return nil
}

func noneEmpty(vals ...string) bool {
	for _, val := range vals {
		if val == "" {
			return false
		}
	}
	return true
}

func deliverSignup(writer http.ResponseWriter, userfill, errorText string) {
	if errorText != "" {
		errorText = "\n" + genErrorForm(errorText) // newline to push the error down
	}
	userfill = genUserForm(userfill)
	startForm("Sign Up", writer)
	insertStylesheet("login", writer)
	insertScript("signup", writer)
	writer.Write([]byte(fmt.Sprintf(`
<form name="create_account" action="signup" method="post" accept-charset="utf-8">%s
	<label>Create Account</label>
	<ul>
		<li>%s</li>
		<li><input name="realname" type="text" placeholder="Full Name" required></li>
		<li><input name="team" type="number" placeholder="Team Number" required></li>
		<li>%s</li>
		<li><input name="passwordconf" type="password" placeholder="Confirm Password" required></li>
		<li><input type="submit" value="Create Account" required><span>Accounts must be recreated at the beginning of every year</span></li>
	</ul>
</form>`, errorText, userfill, frmPassword)))
}

func signupHandler(writer http.ResponseWriter, request *http.Request) error {
	if user := server.GetUser(request); user != nil {
		// if the user is already logged in, redirect to the main page
		http.Redirect(writer, request, "/", http.StatusFound)
		return nil
	}

	if request.Method == "POST" {
		err := request.ParseForm()
		if err != nil {
			deliverSignup(writer, "", "")
			server.HandlerLog.Println(data.HTTPRequestParseError{Request: request}.Error())
			return nil
		}

		fmt.Println(request.PostForm)
		username := request.PostFormValue("username")
		realname := request.PostFormValue("realname")
		password := request.PostFormValue("password")
		passwordconf := request.PostFormValue("passwordconf")
		team, err := strconv.ParseInt(request.PostFormValue("team"), 10, 16)
		if err != nil || team <= 0 {
			deliverSignup(writer, username, "The team number was not valid (make sure it's greater than 0)")
			return nil
		}
		if username == "" || realname == "" || password == "" || passwordconf == "" {
			deliverSignup(writer, username, "The username, real name, password, or password confirmation was empty")
			return nil
		}

		authCookie, err := server.CreateUser(username, realname, password, int(team))
		if err != nil {
			switch err.(type) {
			case data.UsernameTakenError:
				deliverSignup(writer, username, genErrorForm("That username is already taken"))
				return nil
			default:
				deliverSignup(writer, username, "an unknown error occured")
				server.HandlerLog.Printf("unknown signup error (%s)\n", err.Error())
				return nil
			}
		}
		http.SetCookie(writer, authCookie)
		http.Redirect(writer, request, "/matches", http.StatusFound)
		return nil
	} else if request.Method == "GET" {
		deliverSignup(writer, "", "")
		return nil
	}
	return data.HTTPMethodError{Method: request.Method}
}

func logoutHandler(writer http.ResponseWriter, request *http.Request) error {
	http.Redirect(writer, request, "/", http.StatusFound)
	return nil
}

func teamsHandler(writer http.ResponseWriter, request *http.Request) error {
	return data.HTTPNotFoundError{URL: request.URL}
}

func allDataHandler(writer http.ResponseWriter, request *http.Request) error {
	return data.HTTPNotFoundError{URL: request.URL}
}

func matchesHandler(writer http.ResponseWriter, request *http.Request) error {
	return data.HTTPNotFoundError{URL: request.URL}
}

func submissionHandler(writer http.ResponseWriter, request *http.Request) error {
	return data.HTTPNotFoundError{URL: request.URL}
}

func fileHandler(writer http.ResponseWriter, request *http.Request) error {
	return data.HTTPNotFoundError{URL: request.URL}
}

func resourceHandler(writer http.ResponseWriter, request *http.Request) error {
	if strings.HasPrefix(request.URL.Path, "/js") {
		request.URL.Path += ".js"
		writer.Header().Set("Content-Type", "text/javascript")
	} else if strings.HasPrefix(request.URL.Path, "/css") {
		request.URL.Path += ".css"
		writer.Header().Set("Content-Type", "text/css")
	} else {
		return data.HTTPNotFoundError{URL: request.URL}
	}

	f, err := os.Open(filepath.Join(".", request.URL.Path))
	if err != nil {
		return data.HTTPNotFoundError{URL: request.URL}
	}
	defer f.Close()
	io.Copy(writer, f)
	return nil
}
