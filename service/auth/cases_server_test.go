package auth_test

import (
	"net/http"
)

type want struct {
	body       string
	statusCode int
	logMessage string
}
type user struct {
	login    string
	password string
}

var examplesJWTServerIssueTokenNewUser = []struct {
	name        string
	storage     bool
	initial     []*user
	requestForm map[string]string
	want
}{
	{
		"OK",
		true,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"login": "login3", "password": "password3"},
		want{
			statusCode: http.StatusCreated,
		},
	},
	{
		"Bad storage",
		false,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password1"},
		want{
			body:       `{"error":"Internal server error"}`,
			statusCode: http.StatusInternalServerError,
			logMessage: "storage error",
		},
	},
	{
		"No password",
		true,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"login": "login3"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnprocessableEntity,
			logMessage: "",
		},
	},
	{
		"No login",
		true,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"password": "password3"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnprocessableEntity,
			logMessage: "",
		},
	},
	{
		"Login taken",
		true,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password3"},
		want{
			body:       `{"error":"Login already taken"}`,
			statusCode: http.StatusConflict,
			logMessage: "",
		},
	},
}

var examplesJWTServerIssueTokenExistingUser = []struct {
	name        string
	storage     bool
	initial     []*user
	requestForm map[string]string
	want
}{
	{
		"OK",
		true,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password1"},
		want{
			statusCode: http.StatusCreated,
		},
	},
	{
		"Bad storage",
		false,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password1"},
		want{
			body:       `{"error":"Internal server error"}`,
			statusCode: http.StatusInternalServerError,
			logMessage: "storage error",
		},
	},
	{
		"No password",
		true,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"login": "login1"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
	{
		"No login",
		true,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"password": "password1"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
	{
		"Login not in storage",
		true,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"login": "login3", "password": "password3"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
	{
		"Wrong passwords",
		true,
		[]*user{&user{password: "password1", login: "login1"}, &user{password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password3"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
}
