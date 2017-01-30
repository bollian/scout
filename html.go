package main

import (
	"fmt"
	"net/http"
)

const (
	frmLoginButtons = `
<a id="login-link">
	<div id="login-box" href="/login">
		Log In
	</div>
</a><a id="signup-link">
	<div id="signup-box" href="/signup">
		Sign Up
	</div>
<a/>`
)

func genPageStart(title string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<title>%s</title>
	</head>
	<body>`, title)
}

func genPageEnd() string {
	return "</body></html>"
}

func genStylesheetElement(name string) string {
	return fmt.Sprintf(`<link href="%s.css" rel="stylesheet" type="text/css"/>`, name)
}

func genScriptElement(name string) string {
	return fmt.Sprintf(`<script src="%s.js" type="text/javascript"></script>`, name)
}

func genTopBar(request *http.Request) string {
	return fmt.Sprintf(`
%s
<div class="top-bar-container">
	<div class="top-bar-horizontal">
		<a href="/">some icon</a>
		<span class="top-bar-title">εΔ Scout</span>
		<div class="top-bar-user">some user</div>
	</div>
</div>
<div class="top-bar-spacer"></div>`, genStylesheetElement("topbar"))
}

func genTextInput(name, value, placeholder string, required bool) string {
	if value != "" {
		value = fmt.Sprintf(`value="%s"`, value)
	}
	requiredAttribute := ""
	if required {
		requiredAttribute = "required"
	}
	return fmt.Sprintf(`<input name="%s" %s placeholder="%s" type="text" %s>`, name, value, placeholder, requiredAttribute)
}

func genPasswordForm(name string, placeholder string) string {
	return fmt.Sprintf(`<input name="%s" placeholder="%s" type="password">`, name, placeholder)
}

func genTeamNumberForm(value int) string {
	valueAttribute := ""
	if value > 0 {
		valueAttribute = fmt.Sprintf(`value="%d"`, value)
	}
	return fmt.Sprintf(`<input name="team-number" %s placeholder="Team Number" type="number" min="1" step="1">`, valueAttribute)
}
