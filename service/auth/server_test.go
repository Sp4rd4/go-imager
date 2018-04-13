package auth_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/sp4rd4/go-imager/service/auth"
	"github.com/sp4rd4/go-imager/util"
	"github.com/stretchr/testify/assert"
)

func TestNewJWTServer(t *testing.T) {

	examples := []struct {
		name    string
		db      auth.Storage
		secret  []byte
		options []auth.Option
		wantErr error
	}{
		{"OK", &auth.DB{}, []byte("secret"), []auth.Option{auth.WithRequestIDKey("key")}, nil},
		{"Empty secret", &auth.DB{}, nil, []auth.Option{}, errors.New("no secret")},
		{"Nil DB", nil, []byte("secret"), []auth.Option{}, errors.New("missing storage")},
	}
	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			_, err := auth.NewJWTServer(ex.db, ex.secret, ex.options...)
			assert.EqualValues(t, ex.wantErr, err)
		})
	}
}

func TestJWTServerWithRequestIDKey(t *testing.T) {
	examples := []struct {
		name    string
		key     util.RequestKey
		wantErr error
	}{
		{"Empty key", "", errors.New("key is empty")},
		{"OK", "key", nil},
	}
	for _, ex := range examples {
		js := &auth.JWTServer{}
		t.Run(ex.name, func(t *testing.T) {
			assert.EqualValues(t, ex.wantErr, auth.WithRequestIDKey(ex.key)(js), "Expected different error")
		})
	}
}

func TestJWTServerWithLogger(t *testing.T) {
	examples := []struct {
		name    string
		logger  *log.Logger
		wantErr error
	}{
		{"Nil logger", nil, errors.New("logger is missing")},
		{"OK", log.New(), nil},
	}
	for _, ex := range examples {
		js := &auth.JWTServer{}
		t.Run(ex.name, func(t *testing.T) {
			assert.EqualValues(t, ex.wantErr, auth.WithLogger(ex.logger)(js), "Expected different error")
		})
	}
}

func TestJWTServerWithExpiration(t *testing.T) {
	examples := []struct {
		name       string
		expiration time.Duration
		wantErr    error
	}{
		{"Zero expiration", 0, errors.New("expiration duration is zero")},
		{"OK", time.Hour, nil},
	}
	for _, ex := range examples {
		js := &auth.JWTServer{}
		t.Run(ex.name, func(t *testing.T) {
			assert.EqualValues(t, ex.wantErr, auth.WithExpiration(ex.expiration)(js), "Expected different error")
		})
	}
}

type stubStoreSlice struct {
	users []*auth.User
	up    bool
}

func (ss *stubStoreSlice) CreateUser(u *auth.User) (err error) {
	if !ss.up {
		err = errors.New("storage error")
	}
	u.ID = uint64(len(ss.users)) + 1
	ss.users = append(ss.users, u)
	return
}

func (ss *stubStoreSlice) LoadUserByLogin(u *auth.User) (err error) {
	if !ss.up {
		err = errors.New("storage error")
	}
	for _, usr := range ss.users {
		if u.Login == usr.Login {
			*u = *usr
			return
		}
	}
	return sql.ErrNoRows
}

//  refactor similarity
func TestJWTServerIssueTokenNewUser(t *testing.T) {
	log, hook := test.NewNullLogger()
	secret := []byte("verysecret")
	issuer := "verycool"
	expire := time.Hour
	for _, ex := range examplesJWTServerIssueTokenNewUser {
		storage := &stubStoreSlice{[]*auth.User{}, ex.storage}
		js, err := auth.NewJWTServer(
			storage,
			secret,
			auth.WithLogger(log),
			auth.WithIssuer(issuer),
			auth.WithExpiration(expire),
		)
		if err != nil {
			t.Fatal(err)
		}

		prepareStorage(t, ex.initial, storage)
		req := generateRequest(t, ex.requestForm)
		w := httptest.NewRecorder()

		t.Run(ex.name, func(t *testing.T) {
			js.IssueTokenNewUser(w, req)
			if ex.body == "" {
				assertOKResponse(t, w.Body.Bytes(), secret, issuer)
				return
			}
			assertBadResponse(t, hook, w, ex.want)
			hook.Reset()
		})
	}
}

func TestJWTServerIssueTokenExistingUser(t *testing.T) {
	log, hook := test.NewNullLogger()
	secret := []byte("verysecret")
	issuer := "verycool"
	expire := time.Hour
	for _, ex := range examplesJWTServerIssueTokenExistingUser {
		storage := &stubStoreSlice{[]*auth.User{}, ex.storage}
		js, err := auth.NewJWTServer(
			storage,
			secret,
			auth.WithLogger(log),
			auth.WithIssuer(issuer),
			auth.WithExpiration(expire),
		)
		if err != nil {
			t.Fatal(err)
		}

		prepareStorage(t, ex.initial, storage)
		req := generateRequest(t, ex.requestForm)
		w := httptest.NewRecorder()

		t.Run(ex.name, func(t *testing.T) {
			js.IssueTokenExistingUser(w, req)
			if ex.body == "" {
				assertOKResponse(t, w.Body.Bytes(), secret, issuer)
				return
			}
			assertBadResponse(t, hook, w, ex.want)
			hook.Reset()
		})
	}
}

func prepareStorage(t *testing.T, initial []*user, storage auth.Storage) {
	for _, usr := range initial {
		hash, err := auth.HashPassword(usr.password)
		if err != nil {
			t.Fatal(err)
		}
		storage.CreateUser(&auth.User{Login: usr.login, PasswordHash: hash})
	}
}

func generateRequest(t *testing.T, formVals map[string]string) *http.Request {
	form := url.Values{}
	for k, v := range formVals {
		form.Set(k, v)
	}
	body := strings.NewReader(form.Encode())
	req, err := http.NewRequest("POST", "", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func assertOKResponse(t *testing.T, b, secret []byte, issuer string) {
	resp := &auth.Token{}
	if err := json.Unmarshal(b, resp); err != nil {
		t.Fatal(err)
	}
	token, err := jwt.ParseWithClaims(resp.Token, &util.AuthTokenClaims{}, func(tkn *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if assert.Nil(t, err, "No error on parsing should be present") {
		claims, ok := token.Claims.(*util.AuthTokenClaims)
		if assert.True(t, ok, "Claims shoud meet required type") && assert.True(t, token.Valid, "Token should be valid") {
			assert.True(t, claims.VerifyExpiresAt(time.Now().Unix(), true), "Token shouldn't be expired")
			assert.True(t, claims.VerifyIssuer(issuer, true), "Token should have correct issuer")
			assert.NotZero(t, claims.ID, "ID claim should be above zero")
		}
	}
}

func assertBadResponse(t *testing.T, hook *test.Hook, w *httptest.ResponseRecorder, want want) {
	if len(want.logMessage) > 0 && assert.Equal(t, 1, len(hook.Entries), "Should have log entry") {
		assert.Regexp(
			t,
			regexp.MustCompile(want.logMessage),
			hook.Entries[0].Message,
			"Incorrect log entry message",
		)
	}
	assert.Equal(t, want.statusCode, w.Result().StatusCode, "Incorrect response code")
	b, err := ioutil.ReadAll(w.Result().Body)
	w.Result().Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	assert.Regexp(t, want.body, string(b), "Incorrect response body")
}
