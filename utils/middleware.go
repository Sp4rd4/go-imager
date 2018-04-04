package utils

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/oklog/ulid"
)

const RequestGUIDKey = "guid"
const RequestUserKey = "user"

type AuthTokenClaims struct {
	Login string
	Id    uint64
	jwt.StandardClaims
}

func RequestGUID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		entropy := rand.New(rand.NewSource(now.UnixNano()))
		ulid := ulid.MustNew(ulid.Timestamp(now), entropy)
		ctx := context.WithValue(r.Context(), RequestGUIDKey, ulid.String())

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Logger(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			guid, _ := r.Context().Value(RequestGUIDKey).(string)
			requestLogger := logger.WithFields(log.Fields{
				"request_id":  guid,
				"method":      r.Method,
				"url":         r.URL.String(),
				"remote_addr": r.RemoteAddr,
			})

			timeStart := time.Now()
			requestLogger.WithFields(log.Fields{"time": timeStart.Format(time.RFC3339)}).
				Info("Received")

			next.ServeHTTP(w, r)

			timeEnd := time.Now()
			requestLogger.WithFields(log.Fields{"time": timeEnd.Format(time.RFC3339)}).
				Infof("Finished in %s", timeEnd.Sub(timeStart))
		})
	}
}

func CheckJWT(secret []byte, issuer string, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := fromAuthHeader(r)
			if err != nil {
				jwtErrHandler(w, r, logger, err)
				return
			}

			token, err := jwt.ParseWithClaims(tokenStr, &AuthTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
				return secret, nil
			})
			if err != nil {
				jwtErrHandler(w, r, logger, err)
				return
			}

			if method, ok := token.Header["alg"].(string); !ok || method != jwt.SigningMethodHS256.Alg() {
				jwtErrHandler(w, r, logger, fmt.Errorf("Instead of %s token specified %s signing method",
					jwt.SigningMethodHS256.Alg(),
					token.Header["alg"]))
				return
			}
			if err = checkTokenWithClaims(token, issuer); err != nil {
				jwtErrHandler(w, r, logger, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), RequestUserKey, token)))
		})
	}
}

func fromAuthHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization Token is missing")
	}

	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", errors.New("Authorization header format must be Bearer {token}")
	}

	return authHeaderParts[1], nil
}

func jwtErrHandler(w http.ResponseWriter, r *http.Request, logger *log.Logger, err error) {
	guid, _ := r.Context().Value(RequestGUIDKey).(string)
	logger.WithFields(log.Fields{
		"request_id": guid,
	}).Info(err)

	JsonResponse(w, http.StatusUnauthorized, `{"error":"Bad credentials"}`)
}

func checkTokenWithClaims(token *jwt.Token, issuer string) error {
	if !token.Valid {
		return errors.New("Token is invalid")
	}

	claims, ok := token.Claims.(*AuthTokenClaims)
	if !ok {
		return errors.New("Token claims is invalid")
	}
	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return errors.New("Token is expired")
	}
	if !claims.VerifyIssuer(issuer, true) {
		return errors.New("Token issuer is wrong")
	}
	if !(claims.Id > 0) {
		return errors.New("Token Id is nil")
	}

	return nil
}
