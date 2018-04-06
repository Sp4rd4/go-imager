package utils_test

import (
	"context"
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

func TestRequestGUID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		guid, _ := r.Context().Value(utils.RequestGUIDKey).(string)
		_, err := ulid.Parse(guid)
		assert.Nil(t, err)
	})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	log, _ := test.NewNullLogger()
	utils.RequestGUID(log)(handler).ServeHTTP(w, req)
}

func TestLogger(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req.RemoteAddr = "localhost"
	ctx := context.WithValue(req.Context(), utils.RequestGUIDKey, "guid")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	log, hook := test.NewNullLogger()
	utils.Logger(log)(handler).ServeHTTP(w, req)

	assert := assert.New(t)
	if assert.Equal(2, len(hook.Entries)) {
		assert.Equal(logrus.InfoLevel, hook.Entries[0].Level)
		assert.Equal("guid", hook.Entries[0].Data["request_id"])
		assert.Equal("GET", hook.Entries[0].Data["method"])
		assert.Equal("http://example.com/foo", hook.Entries[0].Data["url"])
		assert.Equal("localhost", hook.Entries[0].Data["remote_addr"])
		assert.Equal("Received", hook.Entries[0].Message)

		assert.Equal(logrus.InfoLevel, hook.Entries[1].Level)
		assert.Equal("guid", hook.Entries[1].Data["request_id"])
		assert.Equal("GET", hook.Entries[1].Data["method"])
		assert.Equal("http://example.com/foo", hook.Entries[1].Data["url"])
		assert.Equal("localhost", hook.Entries[1].Data["remote_addr"])
		assert.Regexp(regexp.MustCompile("Finished in "), hook.Entries[1].Message)
	}
}

func TestCheckJWT(t *testing.T) {
	log, hook := test.NewNullLogger()
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	ctx := context.WithValue(req.Context(), utils.RequestGUIDKey, "guid")
	req = req.WithContext(ctx)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	assert := assert.New(t)
	for _, ex := range examplesJWT {
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
		if len(ex.result.logMessage) > 0 && assert.Equal(1, len(hook.Entries)) {
			assert.Regexp(regexp.MustCompile(ex.result.logMessage), hook.Entries[0].Message)
		}

		assert.Equal(ex.result.statusCode, w.Result().StatusCode)
		b, err := ioutil.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}
		assert.Regexp(ex.result.body, string(b))
		hook.Reset()
	}

}
func generateToken(expires int64, issuer, secret string, id uint64) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = &utils.AuthTokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expires,
			Issuer:    issuer,
		},
		Id:    id,
		Login: "login",
	}
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenString, nil

}
