package server

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/oklog/ulid"
	"github.com/sp4rd4/go-imager/services/images/db"
)

const RequestUserIdKey = 0
const RequestGUIDKey = 1

type ImageServer interface {
	ListImages(w http.ResponseWriter, r *http.Request)
	PostImage(w http.ResponseWriter, r *http.Request)
}

type LocalImageServer struct {
	db                db.Storage
	staticsFolderPath string
	log               *log.Logger
}

type imageData struct {
	filename string
	data     []byte
}

func NewLocalImageServer(db db.Storage, staticsFolderPath string, log *log.Logger) (ImageServer, error) {
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
	userId, ok := ctx.Value(RequestUserIdKey).(int)
	if !ok {
		jsonResponse(w, http.StatusUnprocessableEntity, `{"error":"No user_id provided"}`)
		return
	}

	guid, _ := ctx.Value(RequestGUIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": guid})

	id, err := extractImage(r, "image")
	if err != nil {
		requestLogger.Info(err)
		jsonResponse(w, http.StatusUnprocessableEntity, `{"error":"No image is present"}`)
		return
	}

	now := time.Now()
	entropy := rand.New(rand.NewSource(now.UnixNano()))
	ulid, err := ulid.New(ulid.Timestamp(now), entropy)
	if err != nil {
		requestLogger.Warn(err)
		jsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}

	filename := ulid.String() + id.filename
	err = ioutil.WriteFile(filepath.Join(is.staticsFolderPath, filename), id.data, 0644)
	if err != nil {
		requestLogger.Warn(err)
		jsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}

	image := &db.Image{
		Filename: filename,
		UserId:   userId,
	}
	err = is.db.InsertImage(image)
	if err != nil {
		requestLogger.Warn(err)
		jsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}
	w.WriteHeader(http.StatusCreated)
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

func (is *LocalImageServer) ListImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userId, ok := ctx.Value(RequestUserIdKey).(int)
	if !ok {
		jsonResponse(w, http.StatusUnprocessableEntity, `{"error":"No user_id provided"}`)
		return
	}

	guid, _ := ctx.Value(RequestGUIDKey).(string)
	requestLogger := is.log.WithFields(log.Fields{"request_id": guid})

	params := r.URL.Query()
	var limit, offset int
	limitPars, ok := params["limit"]
	if ok && len(limitPars) > 0 {
		limit, _ = strconv.Atoi(params["limit"][0])
	} else {
		limit = 0
	}
	offsetPars, ok := params["offset"]
	if ok && len(offsetPars) > 0 {
		offset, _ = strconv.Atoi(params["offset"][0])
	} else {
		offset = 0
	}
	images, err := is.db.SelectImages(userId, limit, offset)
	if err != nil {
		requestLogger.Warn(err)
		jsonResponse(w, http.StatusInternalServerError, `{"error":"Internal error occurred"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(images)
}

func jsonResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	io.WriteString(w, message)
}
