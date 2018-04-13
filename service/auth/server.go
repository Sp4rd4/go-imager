package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
	"github.com/sp4rd4/go-imager/util"
	"golang.org/x/crypto/bcrypt"
)

// PasswordHashCost defines password hash ing cost for bcrypt
const PasswordHashCost = 10

// TokenServer defines interface for service that can issues authorization tokens for users.
type TokenServer interface {
	IssueTokenNewUser(w http.ResponseWriter, r *http.Request)
	IssueTokenExistingUser(w http.ResponseWriter, r *http.Request)
}

// JWTServer is TokenServer that issues JWT.
type JWTServer struct {
	storage         Storage
	log             *log.Logger
	secret          []byte
	tokenExpiration time.Duration
	issuer          string
	requestIDKey    util.RequestKey
}

// Token describes JWTServer response json
type Token struct {
	TokenType string `json:"token_type"`
	Token     string `json:"access_token"`
}

// Option describes option for JWTServer initializer
type Option func(*JWTServer) error

// NewJWTServer is initializer with functional options for JWTServer.
func NewJWTServer(storage Storage, secret []byte, options ...Option) (TokenServer, error) {
	if storage == nil {
		return nil, errors.New("missing storage")
	}
	if len(secret) == 0 {
		return nil, errors.New("no secret")
	}
	log.SetOutput(os.Stdout)
	log := log.New()
	duration, err := time.ParseDuration("12h")
	if err != nil {
		return nil, err
	}
	js := &JWTServer{storage, log, secret, duration, "", util.RequestIDKey}
	for _, option := range options {
		if err := option(js); err != nil {
			return nil, err
		}
	}
	return js, nil
}

// WithRequestIDKey is functional option for setting JWTServer request_id context key,
// default key is util.RequestIDKey.
func WithRequestIDKey(key util.RequestKey) Option {
	return func(js *JWTServer) error {
		if key == "" {
			return errors.New("key is empty")
		}
		js.requestIDKey = key
		return nil
	}
}

// WithLogger is functional option for setting JWTServer logger
func WithLogger(logger *log.Logger) Option {
	return func(js *JWTServer) error {
		if logger == nil {
			return errors.New("logger is missing")
		}
		js.log = logger
		return nil
	}
}

// WithIssuer is functional option for setting JWTServer issuer
func WithIssuer(issuer string) Option {
	return func(js *JWTServer) error {
		js.issuer = issuer
		return nil
	}
}

// WithExpiration is functional option for setting JWTServer token expiration
func WithExpiration(expire time.Duration) Option {
	return func(js *JWTServer) error {
		if expire == 0 {
			return errors.New("expiration duration is zero")
		}
		js.tokenExpiration = expire
		return nil
	}
}

// IssueTokenNewUser creates user storage record and issues new token for that user
// * login (required) POST form value
// * password (required) POST form value
func (js *JWTServer) IssueTokenNewUser(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(util.RequestIDKey).(string)
	requestLogger := js.log.WithFields(log.Fields{"request_id": requestID})

	if r.FormValue("login") == "" || r.FormValue("password") == "" {
		util.JSONResponse(w, http.StatusUnprocessableEntity, `{"error":"Bad credentials"}`, requestLogger)
		return
	}
	user := &User{Login: r.FormValue("login")}
	err := js.storage.LoadUserByLogin(user)
	if err == nil {
		util.JSONResponse(w, http.StatusConflict, `{"error":"Login already taken"}`, requestLogger)
		return
	}
	if err != sql.ErrNoRows {
		requestLogger.Error(err)
		util.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}

	hash, err := HashPassword(r.FormValue("password"))
	if err != nil {
		requestLogger.Error(err)
		util.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}

	user = &User{
		Login:        r.FormValue("login"),
		PasswordHash: hash,
	}
	if err = js.storage.CreateUser(user); err != nil {
		if _, ok := err.(ErrUniqueIndexConflict); ok {
			util.JSONResponse(w, http.StatusConflict, `{"error":"Login already taken"}`, requestLogger)
			return
		}
		requestLogger.Error(err)
		util.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}

	if err = js.reponseJWTToken(user, w); err != nil {
		requestLogger.Error(err)
		util.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}
}

// IssueTokenExistingUser issues new token for existing user, cheking his password and login
// * login (required) POST form value
// * password (required) POST form value
func (js *JWTServer) IssueTokenExistingUser(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(util.RequestIDKey).(string)
	requestLogger := js.log.WithFields(log.Fields{"request_id": requestID})

	if r.FormValue("password") == "" || r.FormValue("login") == "" {
		util.JSONResponse(w, http.StatusUnauthorized, `{"error":"Bad credentials"}`, requestLogger)
		return
	}

	user := &User{Login: r.FormValue("login")}
	err := js.storage.LoadUserByLogin(user)
	if err == sql.ErrNoRows || !CheckPasswordHash(r.FormValue("password"), user.PasswordHash) {
		util.JSONResponse(w, http.StatusUnauthorized, `{"error":"Bad credentials"}`, requestLogger)
		return
	}
	if err != nil {
		requestLogger.Error(err)
		util.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}

	if err := js.reponseJWTToken(user, w); err != nil {
		requestLogger.Error(err)
		util.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}
}

// HashPassword calculates safe hash of string
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), PasswordHashCost)
	return string(bytes), err
}

// CheckPasswordHash checks whether string and hash are related
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (js *JWTServer) reponseJWTToken(user *User, output http.ResponseWriter) error {
	expiresAt := time.Now().Add(js.tokenExpiration).Unix()
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = &util.AuthTokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
			Issuer:    js.issuer,
		},
		Login: user.Login,
		ID:    user.ID,
	}
	tokenString, err := token.SignedString(js.secret)
	if err != nil {
		return err
	}

	output.Header().Set("Content-Type", "application/json")
	output.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(output).Encode(Token{
		Token:     tokenString,
		TokenType: "Bearer",
	})
	return err
}
