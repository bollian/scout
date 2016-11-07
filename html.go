package main

import (
	"fmt"
	"io"
	"net/http"
)

func startForm(title string, writer io.Writer) {
	writer.Write([]byte(`<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8">
		<title>`))
	writer.Write([]byte(title))
	writer.Write([]byte(`</title>
	</head>
	<body>`))
	insertStylesheet("main", writer)
	insertScript("main", writer)
}

func endForm(writer io.Writer) {
	writer.Write([]byte("</body></html>"))
}

func insertStylesheet(name string, writer io.Writer) {
	writer.Write([]byte("<link href=\"/" + name + ".css\" rel=\"stylesheet\" type=\"text/css\"/>"))
}

func insertScript(name string, writer io.Writer) {
	writer.Write([]byte("<script src=\"/" + name + ".js\" type=\"text/javascript\"></script>"))
}

func insertTopBar(writer io.Writer) {
	insertStylesheet("topbar", writer)
	writer.Write([]byte(`
<div class="top-bar-container">
	<div class="top-bar-horizontal">
		<a href="/">some icon</a>
		<span>some title</span>
		<div>login/signup button</div>
	</div>
</div>
<div class="top-bar-spacer"></div>`))
}

func genErrorForm(msg string) string {
	if msg == "" {
		return msg
	}
	return fmt.Sprintf(`<span class="error-message">%s<br></span>`, msg)
}

func genUserForm(value string) string {
	frm := `<input name="username" type="text" required `
	if value != "" {
		frm += fmt.Sprintf(`value="%s" `, value)
	}
	frm += `placeholder="Username">`
	return frm
}

func redirect(writer http.ResponseWriter) {

}
