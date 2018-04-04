package server

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sp4rd4/go-imager/services/auth/db"
	"golang.org/x/crypto/bcrypt"
)

const RequestGUIDKey = 1

type TokenServer interface {
	IssueTokenNewUser(w http.ResponseWriter, r *http.Request)
	IssueTokenExistingUser(w http.ResponseWriter, r *http.Request)
}

type JWTServer struct {
	db              db.Storage
	log             *log.Logger
	secret          []byte
	tokenExpiration time.Duration
}

type AuthTokenClaims struct {
	*jwt.StandardClaims
	db.User
}

type AuthToken struct {
	TokenType string `json:"token_type"`
	Token     string `json:"access_token"`
}

func NewJWTServer(db db.Storage, log *log.Logger, secret []byte, expiration time.Duration) (TokenServer, error) {
	is := &JWTServer{db, log, secret, expiration}
	return is, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (is *JWTServer) IssueTokenNewUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	guid, _ := ctx.Value(RequestGUIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": guid})

	hash, err := HashPassword(r.FormValue("password"))
	u = &User{
		Login:    r.FormValue("login"),
		Password: hash,
	}
	err = is.db.CreateCredentials(u)
	if err != nil {
		requestLogger.Info(err)
		jsonResponse(w, http.StatusUnprocessableEntity, `{"error":"Wrong credentials"}`)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (is *JWTServer) IssueTokenExistingUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	guid, _ := ctx.Value(RequestGUIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": guid})

	user, err := is.db.LoadUserByLogin(r.FormValue("login"))
	if err != nil {
		requestLogger.Warn(err)
		jsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}
	if user == nil || CheckPasswordHash(r.FormValue("password"), user.PasswordHash) {
		jsonResponse(w, http.StatusUnauthorized, `{"error":"Wrong credentials"}`)
		return
	}

	tokenString, err := createJWTToken(user, is.tokenExpiration, is.secret)
	if err != nil {
		requestLogger.Warn(err)
		jsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthToken{
		Token:     tokenString,
		TokenType: "Bearer",
	})
}

func jsonResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	io.WriteString(w, message)
}

func createJWTToken(user *db.User, expiration time.Duration, secret []byte) (string, error) {
	expiresAt := time.Now().Add(expiration).Unix()
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = &AuthTokenClaims{
		&jwt.StandardClaims{
			ExpiresAt: expiresAt,
		},
		*user,
	}

	return token.SignedString(secret)
}
