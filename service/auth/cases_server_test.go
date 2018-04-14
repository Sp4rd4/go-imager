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

var examplesJWTServerIssueToken = []struct {
	name        string
	new         bool
	storage     bool
	initial     []*user
	requestForm map[string]string
	want
}{
	{
		"New OK",
		true,
		true,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"login": "login3", "password": "password3"},
		want{
			statusCode: http.StatusCreated,
		},
	},
	{
		"New Bad storage",
		true,
		false,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password1"},
		want{
			body:       `{"error":"Internal server error"}`,
			statusCode: http.StatusInternalServerError,
			logMessage: "storage error",
		},
	},
	{
		"New No password",
		true,
		true,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"login": "login3"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnprocessableEntity,
			logMessage: "",
		},
	},
	{
		"New No login",
		true,
		true,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"password": "password3"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnprocessableEntity,
			logMessage: "",
		},
	},
	{
		"New Login taken",
		true,
		true,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password3"},
		want{
			body:       `{"error":"Login already taken"}`,
			statusCode: http.StatusConflict,
			logMessage: "",
		},
	},
	{
		"Existing OK",
		false,
		true,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password1"},
		want{
			statusCode: http.StatusCreated,
		},
	},
	{
		"Existing Bad storage",
		false,
		false,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password1"},
		want{
			body:       `{"error":"Internal server error"}`,
			statusCode: http.StatusInternalServerError,
			logMessage: "storage error",
		},
	},
	{
		"Existing No password",
		false,
		true,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"login": "login1"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
	{
		"Existing No login",
		false,
		true,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"password": "password1"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
	{
		"Existing Login not in storage",
		false,
		true,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"login": "login3", "password": "password3"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
	{
		"Existing Wrong passwords",
		false,
		true,
		[]*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		map[string]string{"login": "login1", "password": "password3"},
		want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
}
