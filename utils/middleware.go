package utils

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/oklog/ulid"
	log "github.com/sirupsen/logrus"
)

// RequestKey is a custom type for a context keys.
type RequestKey string

// RequestIDKey is a context Key for request_id.
const RequestIDKey RequestKey = "requestID"

// RequestUserKey is a context Key for user data.
const RequestUserKey RequestKey = "user"

// AuthTokenClaims are JWT claims for user.
type AuthTokenClaims struct {
	Login string
	ID    uint64
	jwt.StandardClaims
}

// Token allows to implement methods for image.User interface over jwt.Token.
type Token struct {
	*jwt.Token
}

// RequestID middleware adds unique request_id into request context.
func RequestID(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			entropy := rand.New(rand.NewSource(now.UnixNano()))
			ulid, err := ulid.New(ulid.Timestamp(now), entropy)
			if err != nil {
				logger.Error(err)
				JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal Server Error"}`, log.NewEntry(logger))
				return
			}
			ctx := context.WithValue(r.Context(), RequestIDKey, ulid.String())

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Logger middleware logs incoming requests and their timing.
func Logger(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID, _ := r.Context().Value(RequestIDKey).(string)
			remoteAddr := r.RemoteAddr
			if r.Header.Get("X-Real-IP") != "" {
				remoteAddr = r.Header.Get("X-Real-IP")
			}
			requestLogger := logger.WithFields(log.Fields{
				"request_id":  requestID,
				"method":      r.Method,
				"url":         r.URL.String(),
				"remote_addr": remoteAddr,
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

// CheckJWT middleware checks authorization header for JWT token, validates its signature and content,
// then places images.User interface type into context for future use.
func CheckJWT(secret []byte, issuer string, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := fromAuthHeader(r)
			if err != nil {
				jwtErrHandler(w, r, logger, err)
				return
			}

			tkn, err := jwt.ParseWithClaims(tokenStr, &AuthTokenClaims{}, func(tkn *jwt.Token) (interface{}, error) {
				return secret, nil
			})
			if err != nil {
				jwtErrHandler(w, r, logger, err)
				return
			}

			if method, ok := tkn.Header["alg"].(string); !ok || method != jwt.SigningMethodHS256.Alg() {
				jwtErrHandler(w, r, logger, fmt.Errorf("instead of %s token specified %s signing method",
					jwt.SigningMethodHS256.Alg(),
					tkn.Header["alg"]))
				return
			}
			if err = checkTokenWithClaims(tkn, issuer); err != nil {
				jwtErrHandler(w, r, logger, err)
				return
			}
			t := Token{tkn}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), RequestUserKey, &t)))
		})
	}
}

// ID returns user id from context value.
func (tkn *Token) ID() uint64 {
	claims, ok := tkn.Claims.(*AuthTokenClaims)
	if tkn.Valid && ok {
		return claims.ID
	}
	return 0
}

func fromAuthHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization Token is missing")
	}

	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", errors.New("authorization header format must be Bearer {token}")
	}

	return authHeaderParts[1], nil
}

func jwtErrHandler(w http.ResponseWriter, r *http.Request, logger *log.Logger, err error) {
	requestID, _ := r.Context().Value(RequestIDKey).(string)
	requestLogger := logger.WithFields(log.Fields{
		"request_id": requestID,
	})
	requestLogger.Warn(err)
	JSONResponse(w, http.StatusUnauthorized, `{"error":"Bad credentials"}`, requestLogger)
}

func checkTokenWithClaims(token *jwt.Token, issuer string) error {
	if !token.Valid {
		return errors.New("token is invalid")
	}

	claims, ok := token.Claims.(*AuthTokenClaims)
	if !ok {
		return errors.New("token claims is invalid")
	}
	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return errors.New("token is expired")
	}
	if !claims.VerifyIssuer(issuer, true) {
		return errors.New("token issuer is wrong")
	}
	if !(claims.ID > 0) {
		return errors.New("id in token is zero")
	}

	return nil
}
