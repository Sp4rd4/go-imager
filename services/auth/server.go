package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
	"github.com/sp4rd4/go-imager/utils"
	"golang.org/x/crypto/bcrypt"
)

const PasswordHashCost = 10

type TokenServer interface {
	IssueTokenNewUser(w http.ResponseWriter, r *http.Request)
	IssueTokenExistingUser(w http.ResponseWriter, r *http.Request)
}

type JWTServer struct {
	storage         Storage
	log             *log.Logger
	secret          []byte
	tokenExpiration time.Duration
	issuer          string
}

type Token struct {
	TokenType string `json:"token_type"`
	Token     string `json:"access_token"`
}

func NewJWTServer(storage Storage, log *log.Logger, secret []byte, expiration time.Duration, issuer string) (TokenServer, error) {
	if len(secret) == 0 {
		return nil, errors.New("no secret")
	}
	if len(issuer) == 0 {
		return nil, errors.New("no issuer")
	}
	is := &JWTServer{storage, log, secret, expiration, issuer}
	return is, nil
}

func (is *JWTServer) IssueTokenNewUser(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(utils.RequestIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": requestID})

	if r.FormValue("login") == "" || r.FormValue("password") == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, `{"error":"Wrong credentials"}`)
		return
	}
	user := &User{Login: r.FormValue("login")}
	err := is.storage.LoadUserByLogin(user)
	if err == nil {
		utils.JSONResponse(w, http.StatusConflict, `{"error":"Login already taken"}`)
		return
	} else if err != sql.ErrNoRows {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`)
		return
	}

	hash, err := HashPassword(r.FormValue("password"))
	if err != nil {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`)
		return
	}

	user = &User{
		Login:        r.FormValue("login"),
		PasswordHash: hash,
	}

	if err = is.storage.CreateUser(user); err != nil {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`)
		return
	}

	if err = reponseJWTToken(user, is.tokenExpiration, is.issuer, is.secret, w); err != nil {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}
}

func (is *JWTServer) IssueTokenExistingUser(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(utils.RequestIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": requestID})

	user := &User{Login: r.FormValue("login")}
	err := is.storage.LoadUserByLogin(user)
	if err == sql.ErrNoRows || !CheckPasswordHash(r.FormValue("password"), user.PasswordHash) {
		utils.JSONResponse(w, http.StatusUnauthorized, `{"error":"Wrong credentials"}`)
		return
	} else if err != nil {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}

	if err := reponseJWTToken(user, is.tokenExpiration, is.issuer, is.secret, w); err != nil {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), PasswordHashCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func reponseJWTToken(user *User, expiration time.Duration,
	issuer string, secret []byte, output http.ResponseWriter) error {
	expiresAt := time.Now().Add(expiration).Unix()
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = &utils.AuthTokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
			Issuer:    issuer,
		},
		Login: user.Login,
		ID:    user.ID,
	}
	tokenString, err := token.SignedString(secret)
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
