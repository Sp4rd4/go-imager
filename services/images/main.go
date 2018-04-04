package main

import (
	"net/http"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/sp4rd4/go-imager/services/images/db"
	"github.com/sp4rd4/go-imager/services/images/server"
	goji "goji.io"
	"goji.io/pat"
)

func main() {
	dbAddress := os.Getenv("DATABSE_URL")
	staticStoragePath := os.Getenv("STATIC_STORAGE_PATH")
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

	imageServer, err := server.NewLocalImageServer(storage, staticStoragePath, log)
	if err != nil {
		log.Fatal(err)
	}

	mux := goji.NewMux()
	mux.Handle(pat.Get("/images"), wrapMiddleware(imageServer.ListImages, 1))
	mux.Handle(pat.Post("/images"), wrapMiddleware(imageServer.PostImage, 1))

	log.Fatal(http.ListenAndServe(serverHost, mux))
}

func wrapMiddleware(next func(http.ResponseWriter, *http.Request), user_id int) http.Handler {
	return server.RequestGUID(server.CheckJWT(http.HandlerFunc(next), 1))
}
