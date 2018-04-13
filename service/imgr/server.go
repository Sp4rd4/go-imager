package imgr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid"
	log "github.com/sirupsen/logrus"
	"github.com/sp4rd4/go-imager/util"
)

// ImageServer defines interface for service that can list and process uploaded images.
type ImageServer interface {
	ListImages(w http.ResponseWriter, r *http.Request)
	PostImage(w http.ResponseWriter, r *http.Request)
}

// LocalImageServer is ImageServer that stores images in local folder.
type LocalImageServer struct {
	storage        Storage
	staticPath     string
	log            *log.Logger
	requestUserKey utils.RequestKey
	requestIDKey   utils.RequestKey
}

// User interface for getting needed user info from context value.
type User interface {
	ID() uint64
}

// imageData defines uploaded image data.
type imageData struct {
	filename string
	data     []byte
}

// Option describes option for LocalImageServer initializer
type Option func(*LocalImageServer) error

// NewLocalImageServer is initializer with functional options for LocalImageServer.
func NewLocalImageServer(storage Storage, options ...Option) (ImageServer, error) {
	if storage == nil {
		return nil, errors.New("missing storage")
	}
	log.SetOutput(os.Stdout)
	log := log.New()

	is := &LocalImageServer{storage, ".", log, utils.RequestUserKey, utils.RequestIDKey}
	for _, option := range options {
		if err := option(is); err != nil {
			return nil, err
		}
	}
	return is, nil
}

// WithStaticFolder is functional option for setting LocalImageServer static folder path.
func WithStaticFolder(path string) Option {
	return func(is *LocalImageServer) error {
		fileInfo, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !fileInfo.IsDir() {
			return errors.New("path is not pointing to folder")
		}
		is.staticPath = path
		return nil
	}
}

// WithRequestKeys is functional option for setting LocalImageServer request_id  and user context key,
// default keys are utils.RequestIDKey, utils.RequestUserKey.
func WithRequestKeys(user, id utils.RequestKey) Option {
	return func(is *LocalImageServer) error {
		if user == "" || id == "" {
			return errors.New("key is empty")
		}
		if id == user {
			return errors.New("keys are conflicting")
		}
		is.requestIDKey = id
		is.requestUserKey = user
		return nil
	}
}

// WithLogger is functional option for setting LocalImageServer logger
func WithLogger(logger *log.Logger) Option {
	return func(is *LocalImageServer) error {
		if logger == nil {
			return errors.New("logger is missing")
		}
		is.log = logger
		return nil
	}
}

// PostImage saves image to statics folder of LocalImageServer and creates storage record about that image.
// If context value defined by WithRequestUserKey doesn't contain variable
// that implements User interface and returns ID()>0 image will not be processed.
// * image (required) POST multipart-data file
func (is *LocalImageServer) PostImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID, _ := ctx.Value(utils.RequestIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": requestID})

	userID, err := extracrtUserID(ctx)
	if err != nil {
		requestLogger.Warn(err)
		utils.JSONResponse(w, http.StatusUnauthorized, `{"error":"Unauthorized"}`, requestLogger)
		return
	}

	id, err := extractImage(r, requestLogger)
	if err != nil {
		requestLogger.Info(err)
		utils.JSONResponse(w, http.StatusUnprocessableEntity, `{"error":"No image is present"}`, requestLogger)
		return
	}

	now := time.Now()
	entropy := rand.New(rand.NewSource(now.UnixNano()))
	ulid, err := ulid.New(ulid.Timestamp(now), entropy)
	if err != nil {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}

	filename := ulid.String() + id.filename
	if err = ioutil.WriteFile(filepath.Join(is.staticPath, filename), id.data, 0644); err != nil {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}

	image := &Image{
		Filename: filename,
		UserID:   userID,
	}
	if err = is.storage.AddImage(image); err != nil {
		if errU, ok := err.(ErrUniqueIndexConflict); ok {
			requestLogger.Error(errU)
			utils.JSONResponse(w, http.StatusConflict, `{"error":"Image filename is taken"}`, requestLogger)
			return
		}
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// ListImages returns json formatted images list assigned to the current user.
// If context value defined by WithRequestUserKey doesn't contain variable
// that implements User interface and returns ID()>0 no images will be loaded.
// * limit (default: none) Query parameter that states size of returned selection
// * offset (default: 0) Query parameter that states selection offset of returned selection
func (is *LocalImageServer) ListImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID, _ := ctx.Value(utils.RequestIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": requestID})
	userID, err := extracrtUserID(ctx)
	if err != nil {
		requestLogger.Warn(err)
		utils.JSONResponse(w, http.StatusUnauthorized, `{"error":"Unauthorized"}`, requestLogger)
		return
	}

	params := r.URL.Query()
	limit, _ := strconv.ParseUint(params.Get("limit"), 10, 64)
	offset, _ := strconv.ParseUint(params.Get("offset"), 10, 64)
	images := make([]Image, 0)
	err = is.storage.LoadImages(&images, limit, offset, userID)
	if err != nil && err != sql.ErrNoRows {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(images); err != nil {
		requestLogger.Error(err)
		utils.JSONResponse(w, http.StatusInternalServerError, `{"error":"Internal server error"}`, requestLogger)
		return
	}
}

func extractImage(r *http.Request, log *log.Entry) (*imageData, error) {
	file, info, err := r.FormFile("image")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error(err)
		}
	}()

	bs, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// http.DetectContentType specifies 512 as relevant data for type checking
	if !strings.HasPrefix(http.DetectContentType(bs[:512]), "image/") {
		return nil, errors.New("content type is incorrect")
	}

	return &imageData{info.Filename, bs}, nil
}

func extracrtUserID(ctx context.Context) (uint64, error) {
	if user, ok := ctx.Value(utils.RequestUserKey).(User); ok {
		return user.ID(), nil
	}
	return 0, errors.New("no valid user_id provided")
}
