package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sp4rd4/go-imager/services/auth"
	"github.com/sp4rd4/go-imager/utils"
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

	migrationsFolder := os.Getenv("MIGRATIONS_FOLDER")
	if migrationsFolder == "" {
		var err error
		migrationsFolder, err = filepath.Abs("./db/migrations/")
		if err != nil {
			log.Fatal(err)
		}
	}

	timeouts := []string{
		"TOKEN_EXPIRE",
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

	conn, err := utils.OpenDB(dbAddress, migrationsFolder)
	if err != nil {
		log.Fatal(err)
		utils.CloseAndCheck(conn, log)
	}
	storage := &auth.DB{DB: conn}

	imageServer, err := auth.NewJWTServer(storage, log, []byte(secret), durations["TOKEN_EXPIRE"], issuer)
	if err != nil {
		log.Fatal(err)
	}

	mux := goji.NewMux()
	mux.HandleFunc(pat.Post("/users/sign_in"), imageServer.IssueTokenExistingUser)
	mux.HandleFunc(pat.Post("/users/sign_up"), imageServer.IssueTokenNewUser)
	mux.Use(utils.RequestID(log))
	mux.Use(utils.Logger(log))

	srv := &http.Server{
		ReadTimeout:  durations["HTTP_READ_TIMEOUT"],
		WriteTimeout: durations["HTTP_WRITE_TIMEOUT"],
		IdleTimeout:  durations["HTTP_IDLE_TIMEOUT"],
		Handler:      mux,
		Addr:         serverHost,
	}
	log.Fatal(srv.ListenAndServe())
}
