package images

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
	"github.com/sp4rd4/go-imager/utils"
)

type ImageServer interface {
	ListImages(w http.ResponseWriter, r *http.Request)
	PostImage(w http.ResponseWriter, r *http.Request)
}

type LocalImageServer struct {
	db                Storage
	staticsFolderPath string
	log               *log.Logger
}

type User interface {
	Id() uint64
	Key() string
}

type imageData struct {
	filename string
	data     []byte
}

func NewLocalImageServer(db Storage, staticsFolderPath string, log *log.Logger) (ImageServer, error) {
	fileInfo, err := os.Stat(staticsFolderPath)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, errors.New("staticsFolderPath is not pointing to folder")
	}
	is := &LocalImageServer{db, staticsFolderPath, log}
	return is, nil
}

func (is *LocalImageServer) PostImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	guid, _ := ctx.Value(utils.RequestGUIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": guid})

	userId, err := extracrtUserId(ctx)
	if err != nil {
		requestLogger.Warn(err)
		utils.JsonResponse(w, http.StatusUnauthorized, `{"error":"Unauthorized"}`)
		return
	}

	id, err := extractImage(r, "image")
	if err != nil {
		requestLogger.Info(err)
		utils.JsonResponse(w, http.StatusUnprocessableEntity, `{"error":"No image is present"}`)
		return
	}

	now := time.Now()
	entropy := rand.New(rand.NewSource(now.UnixNano()))
	ulid, err := ulid.New(ulid.Timestamp(now), entropy)
	if err != nil {
		requestLogger.Error(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}

	filename := ulid.String() + id.filename
	err = ioutil.WriteFile(filepath.Join(is.staticsFolderPath, filename), id.data, 0644)
	if err != nil {
		requestLogger.Error(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}

	image := &Image{
		Filename: filename,
		UserId:   userId,
	}
	err = is.db.InsertImage(image)
	if err != nil {
		requestLogger.Error(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (is *LocalImageServer) ListImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	guid, _ := ctx.Value(utils.RequestGUIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": guid})
	userId, err := extracrtUserId(ctx)
	if err != nil {
		requestLogger.Warn(err)
		utils.JsonResponse(w, http.StatusUnauthorized, `{"error":"Unauthorized"}`)
		return
	}

	params := r.URL.Query()
	var limit, offset int
	limitPars, ok := params["limit"]
	if ok && len(limitPars) > 0 {
		limit, _ = strconv.Atoi(params["limit"][0])
	} else {
		limit = 0
	}
	offsetPar, ok := params["offset"]
	if ok && len(offsetPar) > 0 {
		offset, _ = strconv.Atoi(params["offset"][0])
	} else {
		offset = 0
	}
	images := make([]Image, 0)
	err = is.db.SelectImages(&images, limit, offset, userId)
	if err != nil && err != sql.ErrNoRows {
		requestLogger.Error(err)
		utils.JsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(images)
}

func extractImage(r *http.Request, field string) (*imageData, error) {
	file, info, err := r.FormFile(field)
	if err != nil {
		return nil, err
	}

	contentType := info.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, errors.New("Content type is incorrect")
	}

	bs, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	i := &imageData{info.Filename, bs}
	return i, nil
}

func extracrtUserId(ctx context.Context) (uint64, error) {
	var user User
	user, ok := ctx.Value(user.Key()).(User)
	id := user.Id()
	if ok && id > 0 {
		return id, nil
	} else {
		return 0, errors.New("No valid user_id provided")
	}
}
