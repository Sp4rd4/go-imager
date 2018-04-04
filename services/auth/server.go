package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
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

type AuthToken struct {
	TokenType string `json:"token_type"`
	Token     string `json:"access_token"`
}

func NewJWTServer(storage Storage, log *log.Logger, secret []byte, expiration time.Duration, issuer string) (TokenServer, error) {
	if len(secret) == 0 {
		return nil, errors.New("No secret")
	}
	if len(issuer) == 0 {
		return nil, errors.New("No issuer")
	}
	is := &JWTServer{storage, log, secret, expiration, issuer}
	return is, nil
}

func (is *JWTServer) IssueTokenNewUser(w http.ResponseWriter, r *http.Request) {
	guid, _ := r.Context().Value(utils.RequestGUIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": guid})

	if r.FormValue("login") == "" || r.FormValue("password") == "" {
		utils.JsonResponse(w, http.StatusUnprocessableEntity, `{"error":"Wrong credentials"}`)
		return
	}
	user := &User{Login: r.FormValue("login")}
	err := is.storage.LoadUserByLogin(user)
	if err == nil {
		utils.JsonResponse(w, http.StatusConflict, `{"error":"Login already taken"}`)
		return
	} else if err != sql.ErrNoRows {
		requestLogger.Warn(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`)
		return
	}
	err = nil

	hash, err := HashPassword(r.FormValue("password"))
	if err != nil {
		requestLogger.Warn(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`)
		return
	}

	user = &User{
		Login:        r.FormValue("login"),
		PasswordHash: hash,
	}
	err = is.storage.CreateUser(user)
	if err != nil {
		requestLogger.Warn(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`)
		return
	}

	err = reponseJWTToken(user, is.tokenExpiration, is.issuer, is.secret, w)
	if err != nil {
		requestLogger.Warn(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}
}

func (is *JWTServer) IssueTokenExistingUser(w http.ResponseWriter, r *http.Request) {
	guid, _ := r.Context().Value(utils.RequestGUIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": guid})

	user := &User{Login: r.FormValue("login")}
	err := is.storage.LoadUserByLogin(user)
	if err == sql.ErrNoRows || !CheckPasswordHash(r.FormValue("password"), user.PasswordHash) {
		utils.JsonResponse(w, http.StatusUnauthorized, `{"error":"Wrong credentials"}`)
		return
	} else if err != nil {
		requestLogger.Warn(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}

	err = reponseJWTToken(user, is.tokenExpiration, is.issuer, is.secret, w)
	if err != nil {
		requestLogger.Warn(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
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
		Id:    user.Id,
	}
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return err
	}

	output.Header().Set("Content-Type", "application/json")
	output.WriteHeader(http.StatusCreated)
	json.NewEncoder(output).Encode(AuthToken{
		Token:     tokenString,
		TokenType: "Bearer",
	})
	return nil
}
