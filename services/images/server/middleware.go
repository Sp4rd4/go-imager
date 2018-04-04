package server

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid"
)

func RequestGUID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		entropy := rand.New(rand.NewSource(now.UnixNano()))
		ulid := ulid.MustNew(ulid.Timestamp(now), entropy)

		ctx := context.WithValue(r.Context(), RequestGUIDKey, ulid.String())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CheckJWT(next http.Handler, user_id int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, RequestUserIdKey, 1)))
	})
}
