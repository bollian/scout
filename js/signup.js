function signup() {
	var password = document.getElementsByName("password")[0];
	var passwordConf = document.getElementsByName("passwordconf")[0];
	if (password.nodeValue !== passwordConf.nodeValue) {
		password.nodeValue = "";
		passwordConf.nodeValue = "";
		alert("The passwords you entered didn't match.");
	}
}