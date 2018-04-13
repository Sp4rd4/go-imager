package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sp4rd4/go-imager/service/imgr"
	"github.com/sp4rd4/go-imager/util"
	goji "goji.io"
	"goji.io/pat"
)

func main() {
	log.SetOutput(os.Stdout)
	log := log.New()

	dbAddress := os.Getenv("DATABASE_URL")
	serverHost := os.Getenv("HOST")
	secret := os.Getenv("TOKEN_SECRET")
	issuer := os.Getenv("TOKEN_ISSUER")
	staticStoragePath := os.Getenv("STATIC_STORAGE_PATH")

	migrationsFolder := os.Getenv("MIGRATIONS_FOLDER")
	if migrationsFolder == "" {
		var err error
		migrationsFolder, err = filepath.Abs("./migrations/")
		if err != nil {
			log.Fatal(err)
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
			log.Fatal(err)
		}
	}

	conn, err := util.OpenDB(dbAddress, migrationsFolder)
	if err != nil {
		util.CloseAndCheck(conn, log)
		log.Fatal(err)
	}
	storage := &imgr.DB{DB: conn}

	imageServer, err := imgr.NewLocalImageServer(
		storage,
		imgr.WithStaticFolder(staticStoragePath),
		imgr.WithLogger(log),
	)
	if err != nil {
		log.Fatal(err)
	}

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/images"), imageServer.ListImages)
	mux.HandleFunc(pat.Post("/images"), imageServer.PostImage)
	mux.Use(util.RequestID(log))
	mux.Use(util.Logger(log))
	mux.Use(util.CheckJWT([]byte(secret), issuer, log))

	srv := &http.Server{
		ReadTimeout:  durations["HTTP_READ_TIMEOUT"],
		WriteTimeout: durations["HTTP_WRITE_TIMEOUT"],
		IdleTimeout:  durations["HTTP_IDLE_TIMEOUT"],
		Handler:      mux,
		Addr:         serverHost,
	}
	log.Fatal(srv.ListenAndServe())
}
