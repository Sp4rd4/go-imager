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
	newUser     bool
	storage     bool
	initial     []*user
	requestForm map[string]string
	want
}{
	{
		name:        "New OK",
		newUser:     true,
		storage:     true,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"login": "login3", "password": "password3"},
		want: want{
			statusCode: http.StatusCreated,
		},
	},
	{
		name:        "New Bad storage",
		newUser:     true,
		storage:     false,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"login": "login1", "password": "password1"},
		want: want{
			body:       `{"error":"Internal server error"}`,
			statusCode: http.StatusInternalServerError,
			logMessage: "storage error",
		},
	},
	{
		name:        "New No password",
		newUser:     true,
		storage:     true,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"login": "login3"},
		want: want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnprocessableEntity,
			logMessage: "",
		},
	},
	{
		name:        "New No login",
		newUser:     true,
		storage:     true,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"password": "password3"},
		want: want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnprocessableEntity,
			logMessage: "",
		},
	},
	{
		name:        "New Login taken",
		newUser:     true,
		storage:     true,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"login": "login1", "password": "password3"},
		want: want{
			body:       `{"error":"Login already taken"}`,
			statusCode: http.StatusConflict,
			logMessage: "",
		},
	},
	{
		name:        "Existing OK",
		newUser:     false,
		storage:     true,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"login": "login1", "password": "password1"},
		want: want{
			statusCode: http.StatusCreated,
		},
	},
	{
		name:        "Existing Bad storage",
		newUser:     false,
		storage:     false,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"login": "login1", "password": "password1"},
		want: want{
			body:       `{"error":"Internal server error"}`,
			statusCode: http.StatusInternalServerError,
			logMessage: "storage error",
		},
	},
	{
		name:        "Existing No password",
		newUser:     false,
		storage:     true,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"login": "login1"},
		want: want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
	{
		name:        "Existing No login",
		newUser:     false,
		storage:     true,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"password": "password1"},
		want: want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
	{
		name:        "Existing Login not in storage",
		newUser:     false,
		storage:     true,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"login": "login3", "password": "password3"},
		want: want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
	{
		name:        "Existing Wrong passwords",
		newUser:     false,
		storage:     true,
		initial:     []*user{{password: "password1", login: "login1"}, {password: "password2", login: "login2"}},
		requestForm: map[string]string{"login": "login1", "password": "password3"},
		want: want{
			body:       `{"error":"Bad credentials"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "",
		},
	},
}
