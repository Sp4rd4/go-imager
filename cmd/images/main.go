package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sp4rd4/go-imager/services/images"
	"github.com/sp4rd4/go-imager/utils"
	goji "goji.io"
	"goji.io/pat"
)

func main() {
	log.SetOutput(os.Stdout)
	logger := log.New()

	dbAddress := os.Getenv("DATABASE_URL")
	serverHost := os.Getenv("HOST")
	secret := os.Getenv("TOKEN_SECRET")
	issuer := os.Getenv("TOKEN_ISSUER")
	staticStoragePath := os.Getenv("STATIC_STORAGE_PATH")

	migrationsFolder := os.Getenv("MIGRATIONS_FOLDER")
	if migrationsFolder == "" {
		var err error
		migrationsFolder, err = filepath.Abs("./db/migrations/")
		if err != nil {
			logger.Fatal(err)
		}
	}

	timeouts := []string{
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
	}
	durations := make(map[string]time.Duration)
	for _, timeout := range timeouts {
		timeoutVal := os.Getenv(timeout)
		duration, err := time.ParseDuration(timeoutVal)
		durations[timeout] = duration
		if err != nil {
			logger.Fatal(err)
		}
	}

	conn, err := utils.OpenDB(dbAddress, migrationsFolder)
	if err != nil {
		utils.CloseAndCheck(conn, logger)
		log.Fatal(err)
	}
	storage := &images.DB{DB: conn}

	imageServer, err := images.NewLocalImageServer(storage, images.WithStaticFolder(staticStoragePath), images.WithLogger(logger))
	if err != nil {
		log.Fatal(err)
	}

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/images"), imageServer.ListImages)
	mux.HandleFunc(pat.Post("/images"), imageServer.PostImage)
	mux.Use(utils.RequestID(logger))
	mux.Use(utils.Logger(logger))
	mux.Use(utils.CheckJWT([]byte(secret), issuer, logger))

	srv := &http.Server{
		ReadTimeout:  durations["HTTP_READ_TIMEOUT"],
		WriteTimeout: durations["HTTP_WRITE_TIMEOUT"],
		IdleTimeout:  durations["HTTP_IDLE_TIMEOUT"],
		Handler:      mux,
		Addr:         serverHost,
	}
	log.Fatal(srv.ListenAndServe())
}
