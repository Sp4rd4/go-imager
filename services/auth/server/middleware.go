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
