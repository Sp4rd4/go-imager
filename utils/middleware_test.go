package utils_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

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

var examplesJWT = []struct {
	Sent struct {
		secret  string
		issuer  string
		expires int64
		id      uint64
	}
	Server struct {
		secret string
		issuer string
	}
	Result struct {
		body       string
		statusCode int
		logMessage string
		logLevel   string
	}
}{
	{
		{"secret", "me", time.Now() + time.Hour*12, 1},
		{},
		sResult: {},
	},
}

func TestCheckJWT(t *testing.T) {
	log, hook := test.NewNullLogger()
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	ctx := context.WithValue(req.Context(), utils.RequestGUIDKey, "guid")
	ctx := context.WithValue(req.Context(), utils.RequestGUIDKey, "guid")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	utils.Logger(log)(handler).ServeHTTP(w, req)

	assert := assert.New(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
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
func generateToken(expires int64, issuer, secret string, id uint64) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = &utils.AuthTokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expires,
			Issuer:    issuer,
		},
		Id: id,
	}
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil

}
