package util_test

import (
	"net/http"
	"time"
)

type want struct {
	body       string
	statusCode int
	logMessage string
}
type server struct {
	secret string
	issuer string
}
type request struct {
	secret  string
	issuer  string
	expires int64
	id      uint64
}

var examplesJWT = []struct {
	name string
	*request
	server
	want
}{
	{
		"Correct",
		&request{"secret", "me", time.Now().Add(time.Hour).Unix(), 1},
		server{"secret", "me"},
		want{"", http.StatusOK, ""},
	},
	{
		"Incorrect secret",
		&request{"bad_secret", "me", time.Now().Add(time.Hour).Unix(), 1},
		server{"secret", "me"},
		want{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "signature is invalid"},
	},
	{
		"Incorrect issuer",
		&request{"secret", "not_me", time.Now().Add(time.Hour).Unix(), 1},
		server{"secret", "me"},
		want{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "token issuer is wrong"},
	},
	{
		"Expired token",
		&request{"secret", "me", time.Now().Add(-time.Hour).Unix(), 1},
		server{"secret", "me"},
		want{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "token is expired"},
	},
	{
		"Bad user id",
		&request{"secret", "me", time.Now().Add(time.Hour).Unix(), 0},
		server{"secret", "me"},
		want{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "id in token is zero"},
	},
	{
		"Bad authorization header content",
		&request{},
		server{"secret", "me"},
		want{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "authorization header format must be"},
	},
	{
		"No authorization header",
		nil,
		server{"secret", "me"},
		want{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "token contains an invalid number of segments"},
	},
}
