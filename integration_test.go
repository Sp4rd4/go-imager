package integration_test

import (
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Sirupsen/logrus/hooks/test"
	"github.com/jmoiron/sqlx"

	goji "goji.io"
	"goji.io/pat"

	"github.com/sp4rd4/go-imager/service/auth"
	"github.com/sp4rd4/go-imager/service/imgr"
	"github.com/sp4rd4/go-imager/util"
	"gopkg.in/gavv/httpexpect.v1"
)

const imgrSchema = `{
    "$schema": "http://json-schema.org/schema#",
    "type": "array",
    "items": {
        "type": "object",
        "properties": {
            "filename": {
                "type": "string"
            }
        },
        "required": [
            "filename"
        ]
    }
}`

func setupAuthServer(t *testing.T, secret []byte, issuer string) (http.Handler, *sqlx.DB) {
	log, _ := test.NewNullLogger()
	dbAddress := os.Getenv("USERS_DATABASE_URL")
	conn, err := util.OpenDB(dbAddress, "./service/auth/migrations/")
	if err != nil {
		t.Fatal(err)
		util.CloseAndCheck(conn, log)
	}
	storage := &auth.DB{DB: conn}

	tokenServer, err := auth.NewJWTServer(
		storage,
		[]byte(secret),
		auth.WithLogger(log),
		auth.WithExpiration(time.Hour),
		auth.WithIssuer(issuer),
	)
	if err != nil {
		t.Fatal(err)
	}

	mux := goji.NewMux()
	users := goji.SubMux()
	users.HandleFunc(pat.Post("/sign_in"), tokenServer.IssueTokenExistingUser)
	users.HandleFunc(pat.Post("/sign_up"), tokenServer.IssueTokenNewUser)
	users.Use(util.RequestID(log))
	users.Use(util.Logger(log))
	mux.Handle(pat.New("/users/*"), users)

	return mux, conn
}

func setupImageServer(t *testing.T, secret []byte, issuer, staticStorage string) (http.Handler, *sqlx.DB) {
	log, _ := test.NewNullLogger()
	dbAddress := os.Getenv("IMAGES_DATABASE_URL")
	conn, err := util.OpenDB(dbAddress, "./service/imgr/migrations/")
	if err != nil {
		t.Fatal(err)
		util.CloseAndCheck(conn, log)
	}
	storage := &imgr.DB{DB: conn}

	imgrServer, err := imgr.NewLocalImageServer(
		storage,
		imgr.WithStaticFolder(staticStorage),
		imgr.WithLogger(log),
	)
	if err != nil {
		t.Fatal(err)
	}

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/images"), imgrServer.ListImages)
	mux.HandleFunc(pat.Post("/images"), imgrServer.PostImage)
	mux.Use(util.RequestID(log))
	mux.Use(util.Logger(log))
	mux.Use(util.CheckJWT(secret, issuer, log))

	return mux, conn
}
func TestIntegration(t *testing.T) {
	secret := []byte("verysecret")
	issuer := "supercool"

	folder, err := ioutil.TempDir("", "static")
	if err != nil {
		t.Fatal("Unable to create temp dir")
	}
	defer os.RemoveAll(folder)
	authHandler, authDB := setupAuthServer(t, secret, issuer)
	defer util.CleanDB(t, authDB)
	imgHandler, imgrDB := setupImageServer(t, secret, issuer, folder)
	defer util.CleanDB(t, imgrDB)

	authServer := httptest.NewServer(authHandler)
	defer authServer.Close()
	imgrServer := httptest.NewServer(imgHandler)
	defer imgrServer.Close()
	authExpect := httpexpect.New(t, authServer.URL)
	imgrExpect := httpexpect.New(t, imgrServer.URL)

	t.Run("Multiple Users Auth, Upload, List", func(t *testing.T) {
		multipleUsersAuthUploadList(t, authExpect, imgrExpect, authDB, imgrDB)
	})
	t.Run("Single User Auth, Upload, List", func(t *testing.T) {
		singleUserAuthUploadList(t, authExpect, imgrExpect, authDB, imgrDB)
	})
	t.Run("Bad Auth Requests", func(t *testing.T) {
		badAuth(t, authExpect, authDB)
	})
	t.Run("Bad Imgr requests", func(t *testing.T) {
		badImgr(t, authExpect, imgrExpect, authDB, imgrDB)
	})
}

func multipleUsersAuthUploadList(t *testing.T, authExpect, imgrExpect *httpexpect.Expect, authDB, imgrDB *sqlx.DB) {
	defer cleanAuthTable(t, authDB)
	defer cleanImgrTable(t, imgrDB)
	folder, err := ioutil.TempDir("", "upload")
	if err != nil {
		t.Fatal("Unable to create temp dir")
	}
	defer os.RemoveAll(folder)

	// Sign up first user
	authResp := authExpect.POST("/users/sign_up").WithFormField("login", "login1").WithFormField("password", "password1").
		Expect().
		Status(http.StatusCreated).JSON().Object()
	authResp.ContainsKey("token_type").ContainsKey("access_token")
	token := authResp.Value("token_type").String().Raw() + " " + authResp.Value("access_token").String().Raw()

	// Post images for first user
	imgrExpect.POST("/images").WithMultipart().WithFile("image", createImage(t, folder, "login1")).WithHeader("Authorization", token).
		Expect().
		Status(http.StatusCreated).NoContent()
	imgrExpect.POST("/images").WithMultipart().WithFile("image", createImage(t, folder, "login1")).WithHeader("Authorization", token).
		Expect().
		Status(http.StatusCreated).NoContent()

	// Sign up second user
	authResp = authExpect.POST("/users/sign_up").WithFormField("login", "login2").WithFormField("password", "password2").
		Expect().
		Status(http.StatusCreated).JSON().Object()
	authResp.ContainsKey("token_type").ContainsKey("access_token")
	token = authResp.Value("token_type").String().Raw() + " " + authResp.Value("access_token").String().Raw()

	// Post image for second user
	imgrExpect.POST("/images").WithMultipart().WithFile("image", createImage(t, folder, "login2")).WithHeader("Authorization", token).
		Expect().
		Status(http.StatusCreated).NoContent()

	// Sign in first user
	authResp = authExpect.POST("/users/sign_in").WithFormField("login", "login1").WithFormField("password", "password1").
		Expect().
		Status(http.StatusCreated).JSON().Object()
	authResp.ContainsKey("token_type").ContainsKey("access_token")
	token = authResp.Value("token_type").String().Raw() + " " + authResp.Value("access_token").String().Raw()

	// Check first user images
	imgResp := imgrExpect.GET("/images").WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).JSON()
	imgResp.Schema(imgrSchema)
	imgResp.Array().Length().Equal(2)
	for _, img := range imgResp.Path("$..filename").Array().Iter() {
		img.String().Match("login1")
	}

	// Sign in second user
	authResp = authExpect.POST("/users/sign_in").WithFormField("login", "login2").WithFormField("password", "password2").
		Expect().
		Status(http.StatusCreated).JSON().Object()
	authResp.ContainsKey("token_type").ContainsKey("access_token")
	token = authResp.Value("token_type").String().Raw() + " " + authResp.Value("access_token").String().Raw()

	// Check second user images
	imgResp = imgrExpect.GET("/images").WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).JSON()
	imgResp.Schema(imgrSchema)
	imgResp.Array().Length().Equal(1)
	for _, img := range imgResp.Path("$..filename").Array().Iter() {
		img.String().Match("login2")
	}
}

func singleUserAuthUploadList(t *testing.T, authExpect, imgrExpect *httpexpect.Expect, authDB, imgrDB *sqlx.DB) {
	defer cleanAuthTable(t, authDB)
	defer cleanImgrTable(t, imgrDB)
	folder, err := ioutil.TempDir("", "upload")
	if err != nil {
		t.Fatal("Unable to create temp dir")
	}
	defer os.RemoveAll(folder)

	// Sign up user
	authResp := authExpect.POST("/users/sign_up").WithFormField("login", "login1").WithFormField("password", "password1").
		Expect().
		Status(http.StatusCreated).JSON().Object()
	authResp.ContainsKey("token_type").ContainsKey("access_token")
	token := authResp.Value("token_type").String().Raw() + " " + authResp.Value("access_token").String().Raw()

	// Check existing images
	imgResp := imgrExpect.GET("/images").WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).JSON()
	imgResp.Schema(imgrSchema)
	imgResp.Array().Length().Equal(0)

	// Post images for user
	imgrExpect.POST("/images").WithMultipart().WithFile("image", createImage(t, folder, "first")).WithHeader("Authorization", token).
		Expect().
		Status(http.StatusCreated).NoContent()
	imgrExpect.POST("/images").WithMultipart().WithFile("image", createImage(t, folder, "second")).WithHeader("Authorization", token).
		Expect().
		Status(http.StatusCreated).NoContent()
	imgrExpect.POST("/images").WithMultipart().WithFile("image", createImage(t, folder, "third")).WithHeader("Authorization", token).
		Expect().
		Status(http.StatusCreated).NoContent()
	imgrExpect.POST("/images").WithMultipart().WithFile("image", createImage(t, folder, "fourth")).WithHeader("Authorization", token).
		Expect().
		Status(http.StatusCreated).NoContent()

	// Check user images with params
	imgResp = imgrExpect.GET("/images").WithQuery("limit", 2).WithQuery("offset", 1).WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).JSON()
	imgResp.Schema(imgrSchema)
	imgResp.Array().Length().Equal(2)
	for _, img := range imgResp.Path("$..filename").Array().Iter() {
		img.String().Match("(second)|(third)")
	}
}

func badAuth(t *testing.T, authExpect *httpexpect.Expect, authDB *sqlx.DB) {
	defer cleanAuthTable(t, authDB)

	// Sign up user
	authResp := authExpect.POST("/users/sign_up").WithFormField("login", "login1").WithFormField("password", "password1").
		Expect().
		Status(http.StatusCreated).JSON().Object()
	authResp.ContainsKey("token_type").ContainsKey("access_token")

	// Sign up user with duplicated username
	authExpect.POST("/users/sign_up").WithFormField("login", "login1").WithFormField("password", "password2").
		Expect().
		Status(http.StatusConflict).JSON().Object().Value("error").String().Match("Login already taken")

	// Sign in user with wrong password
	authExpect.POST("/users/sign_in").WithFormField("login", "login1").WithFormField("password", "password2").
		Expect().
		Status(http.StatusUnauthorized).JSON().Object().Value("error").String().Match("Bad credentials")

	// Sign up user with empty form
	authExpect.POST("/users/sign_up").
		Expect().
		Status(http.StatusUnprocessableEntity).JSON().Object().Value("error").String().Match("Bad credentials")

	// Sign in user with empty form
	authExpect.POST("/users/sign_in").
		Expect().
		Status(http.StatusUnauthorized).JSON().Object().Value("error").String().Match("Bad credentials")
}

func badImgr(t *testing.T, authExpect, imgrExpect *httpexpect.Expect, authDB, imgrDB *sqlx.DB) {
	defer cleanAuthTable(t, authDB)
	defer cleanImgrTable(t, imgrDB)
	folder, err := ioutil.TempDir("", "upload")
	if err != nil {
		t.Fatal("Unable to create temp dir")
	}
	defer os.RemoveAll(folder)

	// Sign up user
	authResp := authExpect.POST("/users/sign_up").WithFormField("login", "login1").WithFormField("password", "password1").
		Expect().
		Status(http.StatusCreated).JSON().Object()
	authResp.ContainsKey("token_type").ContainsKey("access_token")
	token := authResp.Value("token_type").String().Raw() + " " + authResp.Value("access_token").String().Raw()

	// Post image without authorization header
	imgrExpect.POST("/images").WithMultipart().WithFile("image", createImage(t, folder, "login2")).
		Expect().
		Status(http.StatusUnauthorized).JSON().Object().Value("error").String().Match("Bad credentials")

	// Post image without form
	imgrExpect.POST("/images").WithHeader("Authorization", token).
		Expect().
		Status(http.StatusUnprocessableEntity).JSON().Object().Value("error").String().Match("No image is present")

	// List images without authorization header
	imgrExpect.GET("/images").
		Expect().
		Status(http.StatusUnauthorized).JSON().Object().Value("error").String().Match("Bad credentials")
}

func createImage(t *testing.T, folder, name string) string {
	r := strconv.Itoa(rand.Int())
	f, err := os.Create(filepath.Join(folder, r+name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	img.Set(10, 10, color.RGBA{255, 0, 0, 255})
	if err = png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func cleanAuthTable(t *testing.T, authDB *sqlx.DB) {
	if _, err := authDB.Exec(`TRUNCATE TABLE "users" CASCADE;`); err != nil {
		t.Fatal(err)
	}
}

func cleanImgrTable(t *testing.T, imgrDB *sqlx.DB) {
	if _, err := imgrDB.Exec(`TRUNCATE TABLE "images" CASCADE;`); err != nil {
		t.Fatal(err)
	}
}
