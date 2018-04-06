package utils_test

import (
	"net/http"
	"time"
)

type result struct {
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
	*request
	server
	result
}{
	{
		&request{"secret", "me", time.Now().Add(time.Hour).Unix(), 1},
		server{"secret", "me"},
		result{"", http.StatusOK, ""},
	},
	{
		&request{"bad_secret", "me", time.Now().Add(time.Hour).Unix(), 1},
		server{"secret", "me"},
		result{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "signature is invalid"},
	},
	{
		&request{"secret", "not_me", time.Now().Add(time.Hour).Unix(), 1},
		server{"secret", "me"},
		result{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "Token issuer is wrong"},
	},
	{
		&request{"secret", "me", time.Now().Add(-time.Hour).Unix(), 1},
		server{"secret", "me"},
		result{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "token is expired"},
	},
	{
		&request{"secret", "me", time.Now().Add(time.Hour).Unix(), 0},
		server{"secret", "me"},
		result{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "Id in token is zero"},
	},
	{
		&request{},
		server{"secret", "me"},
		result{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "token contains an invalid number of segments"},
	},
	{
		nil,
		server{"secret", "me"},
		result{`{"error":"Bad credentials"}`, http.StatusUnauthorized, "token contains an invalid number of segments"},
	},
}
