package utils_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"

	"github.com/oklog/ulid"

	"github.com/stretchr/testify/assert"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/sp4rd4/go-imager/utils"
)

func TestRequestID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, _ := r.Context().Value(utils.RequestIDKey).(string)
		_, err := ulid.Parse(requestID)
		assert.Nil(t, err, "There should be valid ulid in request")
	})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	log, _ := test.NewNullLogger()
	utils.RequestID(log)(handler).ServeHTTP(w, req)
}

func TestLogger(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req.RemoteAddr = "localhost"
	ctx := context.WithValue(req.Context(), utils.RequestIDKey, "requestID")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	log, hook := test.NewNullLogger()
	utils.Logger(log)(handler).ServeHTTP(w, req)

	assert := assert.New(t)
	if assert.Equal(2, len(hook.Entries), "There should be valid ulid in request") {
		assert.Equal(logrus.InfoLevel, hook.Entries[0].Level, "Wrong first log entry level")
		assert.Equal("requestID", hook.Entries[0].Data["request_id"], "Wrong first log entry request_id")
		assert.Equal("GET", hook.Entries[0].Data["method"], "Wrong first log entry method")
		assert.Equal("http://example.com/foo", hook.Entries[0].Data["url"], "Wrong first log entry url")
		assert.Equal("localhost", hook.Entries[0].Data["remote_addr"], "Wrong first log entry remoteAddr")
		assert.Equal("Received", hook.Entries[0].Message, "Wrong first log entry message")

		assert.Equal(logrus.InfoLevel, hook.Entries[1].Level, "Wrong second log entry level")
		assert.Equal("requestID", hook.Entries[1].Data["request_id"], "Wrong second log entry request_id")
		assert.Equal("GET", hook.Entries[1].Data["method"], "Wrong second log entry method")
		assert.Equal("http://example.com/foo", hook.Entries[1].Data["url"], "Wrong second log entry url")
		assert.Equal("localhost", hook.Entries[1].Data["remote_addr"], "Wrong second log entry remoteAddr")
		assert.Regexp(regexp.MustCompile("Finished in "), hook.Entries[1].Message, "Wrong second log entry message")
	}
}

func TestCheckJWT(t *testing.T) {
	log, hook := test.NewNullLogger()
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	ctx := context.WithValue(req.Context(), utils.RequestIDKey, "requestID")
	req = req.WithContext(ctx)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	assert := assert.New(t)
	for _, ex := range examplesJWT {
		t.Run(fmt.Sprintf("%+v", ex), func(t *testing.T) {
			var token string
			if ex.request != nil {
				emptyReq := request{}
				if *ex.request == emptyReq {
					token = "RANDOMSTRING"
				} else {
					var err error
					token, err = generateToken(ex.request.expires, ex.request.issuer, ex.request.secret, ex.request.id)
					if err != nil {
						t.Fatal(err)
					}
				}
			}

			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			utils.CheckJWT([]byte(ex.server.secret), ex.server.issuer, log)(handler).ServeHTTP(w, req)
			if len(ex.result.logMessage) > 0 && assert.Equal(1, len(hook.Entries), "Should have log entry") {
				assert.Regexp(
					regexp.MustCompile(ex.result.logMessage),
					hook.Entries[0].Message,
					"Incorrect log entry message",
				)
			}

			assert.Equal(ex.result.statusCode, w.Result().StatusCode, "Incorrect response code")
			b, err := ioutil.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatal(err)
			}
			assert.Regexp(ex.result.body, string(b), "Incorrect response body")

			hook.Reset()
		})
	}

}

func generateToken(expires int64, issuer, secret string, id uint64) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = &utils.AuthTokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expires,
			Issuer:    issuer,
		},
		ID:    id,
		Login: "login",
	}
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenString, nil

}
