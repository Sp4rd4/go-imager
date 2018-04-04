package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/sp4rd4/go-imager/services/auth/db"
	"github.com/sp4rd4/go-imager/services/auth/server"
	goji "goji.io"
	"goji.io/pat"
)

func main() {
	dbAddress := os.Getenv("DATABSE_URL")
	serverHost := os.Getenv("HOST")
	migrationsFolder := os.Getenv("MIGRATIONS_FOLDER")
	var err error
	if migrationsFolder == "" {
		migrationsFolder, err = filepath.Abs("./db/migrations/")
		if err != nil {
			log.Fatal(err)
		}
	}

	log := log.New()
	log.Out = os.Stdout

	var storage db.Storage
	storage, err = db.Open(dbAddress, migrationsFolder)
	if err != nil {
		log.Fatal(err)
	}

	imageServer, err := server.NewJWTServer(storage, log)
	if err != nil {
		log.Fatal(err)
	}

	mux := goji.NewMux()
	mux.Handle(pat.Post("/sign_in"), wrapMiddleware(imageServer.ListImages))
	mux.Handle(pat.Post("/sign_up"), wrapMiddleware(imageServer.PostImage))

	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
		Addr:         serverHost,
	}
	log.Fatal(srv.ListenAndServe())
}

func wrapMiddleware(next func(http.ResponseWriter, *http.Request)) http.Handler {
	return server.RequestGUID(http.HandlerFunc(next))
}
