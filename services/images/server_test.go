package images_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"
	"github.com/sp4rd4/go-imager/services/images"
	"github.com/sp4rd4/go-imager/utils"
)

func TestNewLocalImageServer(t *testing.T) {
	examples := []struct {
		name    string
		db      images.Storage
		options []images.Option
		wantErr error
	}{
		{"OK", &images.DB{}, []images.Option{images.WithRequestUserKey("key")}, nil},
		{
			"Conflicting keys",
			&images.DB{},
			[]images.Option{images.WithRequestUserKey("1"), images.WithRequestIDKey("1")},
			errors.New("key is conflicting"),
		},
		{"Nil DB", nil, []images.Option{}, errors.New("missing storage")},
	}
	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			_, err := images.NewLocalImageServer(ex.db, ex.options...)
			assert.EqualValues(t, ex.wantErr, err)
		})
	}
}

func TestLocalImageServerWithStaticFolder(t *testing.T) {
	folder, err := ioutil.TempDir("", "static")
	if err != nil {
		t.Fatal("Unable to create temp dir")
	}
	defer os.RemoveAll(folder)
	examples := []struct {
		name    string
		path    string
		wantErr error
	}{
		{"Empty path", "", errors.New("stat : no such file or directory")},
		{"Invalid path", "/missingfolder", errors.New("stat /missingfolder: no such file or directory")},
		{"OK", folder, nil},
	}
	for _, ex := range examples {
		is := &images.LocalImageServer{}
		t.Run(ex.name, func(t *testing.T) {
			if ex.wantErr != nil {
				assert.Equal(t, ex.wantErr.Error(), images.WithStaticFolder(ex.path)(is).Error(), "Expected different error")
			} else {
				assert.Nil(t, images.WithStaticFolder(ex.path)(is), "Expected different error")
			}

		})
	}
}

func TestLocalImageServerWithRequestIDKey(t *testing.T) {
	examples := []struct {
		name    string
		key     utils.RequestKey
		wantErr error
	}{
		{"Empty key", "", errors.New("key is empty")},
		{"Conflicting key", utils.RequestUserKey, errors.New("key is conflicting")},
		{"OK", "key", nil},
	}
	for _, ex := range examples {
		is := &images.LocalImageServer{}
		if err := images.WithRequestUserKey(utils.RequestUserKey)(is); err != nil {
			t.Fatal(err)
		}
		t.Run(ex.name, func(t *testing.T) {
			assert.EqualValues(t, ex.wantErr, images.WithRequestIDKey(ex.key)(is), "Expected different error")
		})
	}
}

func TestLocalImageServerWithRequestUserKey(t *testing.T) {
	examples := []struct {
		name    string
		key     utils.RequestKey
		wantErr error
	}{
		{"Empty key", "", errors.New("key is empty")},
		{"Conflicting key", utils.RequestIDKey, errors.New("key is conflicting")},
		{"OK", "key", nil},
	}
	for _, ex := range examples {
		is := &images.LocalImageServer{}
		if err := images.WithRequestIDKey(utils.RequestIDKey)(is); err != nil {
			t.Fatal(err)
		}
		t.Run(ex.name, func(t *testing.T) {
			assert.EqualValues(t, ex.wantErr, images.WithRequestUserKey(ex.key)(is), "Expected different error")
		})
	}
}

func TestLocalImageServerWithLogger(t *testing.T) {
	examples := []struct {
		name    string
		logger  *log.Logger
		wantErr error
	}{
		{"Nil logger", nil, errors.New("logger is missing")},
		{"OK", log.New(), nil},
	}
	for _, ex := range examples {
		is := &images.LocalImageServer{}
		t.Run(ex.name, func(t *testing.T) {
			assert.EqualValues(t, ex.wantErr, images.WithLogger(ex.logger)(is), "Expected different error")
		})
	}
}

type stubStoreNil bool

func (ss stubStoreNil) AddImage(_ *images.Image) (err error) {
	if !ss {
		err = errors.New("storage error")
	}
	return
}
func (ss stubStoreNil) LoadImages(_ *[]images.Image, _, _, _ uint64) (err error) {
	return
}

func TestLocalImageServerPostImage(t *testing.T) {
	logger, hook := test.NewNullLogger()
	for _, ex := range examplesLocalImageServerPostImage {
		staticPath, err := ioutil.TempDir("", "static")
		if err != nil {
			t.Fatal("Unable to create temp dir")
		}
		is, err := images.NewLocalImageServer(
			stubStoreNil(ex.storage),
			images.WithLogger(logger),
			images.WithStaticFolder(staticPath),
		)
		if err != nil {
			t.Fatal(err)
		}

		if ex.staticPath {
			defer os.RemoveAll(staticPath)
		} else {
			os.RemoveAll(staticPath)
		}
		req := generateRequest(t, ex.requestPost.file, ex.requestPost.context)
		w := httptest.NewRecorder()

		t.Run(ex.name, func(t *testing.T) {
			is.PostImage(w, req)
			if len(ex.want.logMessage) > 0 && assert.Equal(t, 1, len(hook.Entries), "Should have log entry") {
				assert.Regexp(
					t,
					regexp.MustCompile(ex.want.logMessage),
					hook.Entries[0].Message,
					"Incorrect log entry message",
				)
			}

			assert.Equal(t, ex.want.statusCode, w.Result().StatusCode, "Incorrect response code")
			b, err := ioutil.ReadAll(w.Result().Body)
			w.Result().Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			assert.Regexp(t, ex.want.body, string(b), "Incorrect response body")

			hook.Reset()
		})
	}
}

func generateRequest(t *testing.T, fileType byte, contextVals map[utils.RequestKey]interface{}) *http.Request {
	var req *http.Request
	var err error
	if fileType == fileMissing {
		req, err = http.NewRequest("POST", "", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		var file *os.File
		body := &bytes.Buffer{}
		multi := multipart.NewWriter(body)
		defer multi.Close()
		file, err = os.Open(createFile(t, fileType))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		defer os.Remove(file.Name())

		fileWriter, err := multi.CreateFormFile("image", file.Name())
		if err != nil {
			t.Fatal(err)
		}

		_, err = io.Copy(fileWriter, file)
		if err != nil {
			t.Fatal(err)
		}
		multi.Close()
		req, err = http.NewRequest("POST", "", body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", multi.FormDataContentType())
	}

	ctx := req.Context()
	for k, v := range contextVals {
		ctx = context.WithValue(ctx, k, v)
	}

	return req.WithContext(ctx)
}

func createFile(t *testing.T, kind byte) string {
	filename := "image.png"
	f, err := os.Create("image.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	switch kind {
	case fileNonImage:
		_, err = f.WriteString("Text file content")
	case fileBroken:
		_, err = f.WriteString("a\xe0\xe5\xf0\xe9\xe1\xf8\xf1\xe9\xe8\xe4Z")
	case fileValid:
		img := image.NewRGBA(image.Rect(0, 0, 50, 50))
		img.Set(10, 10, color.RGBA{255, 0, 0, 255})
		err = png.Encode(f, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return filename
}

type stubStoreSlice bool

func (ss stubStoreSlice) AddImage(_ *images.Image) (err error) {
	return
}
func (ss stubStoreSlice) LoadImages(in *[]images.Image, _, _, _ uint64) (err error) {
	if !ss {
		err = errors.New("storage error")
	} else {
		*in = []images.Image{
			images.Image{Filename: "filename1", UserID: 1},
			images.Image{Filename: "filename2", UserID: 1},
			images.Image{Filename: "filename3", UserID: 1},
		}
	}
	return
}

func TestLocalImageServerListImages(t *testing.T) {
	logger, hook := test.NewNullLogger()
	for _, ex := range examplesLocalImageServerListImages {
		is, err := images.NewLocalImageServer(
			stubStoreSlice(ex.storage),
			images.WithLogger(logger),
		)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("GET", "", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}

		ctx := req.Context()
		for k, v := range ex.requestList.context {
			ctx = context.WithValue(ctx, k, v)
		}

		req = req.WithContext(ctx)
		req.URL.Query().Add("limit", fmt.Sprint(ex.requestList.limit))
		req.URL.Query().Add("offset", fmt.Sprint(ex.requestList.offset))
		w := httptest.NewRecorder()

		t.Run(ex.name, func(t *testing.T) {
			is.ListImages(w, req)
			if len(ex.want.logMessage) > 0 && assert.Equal(t, 1, len(hook.Entries), "Should have log entry") {
				assert.Regexp(
					t,
					regexp.MustCompile(ex.want.logMessage),
					hook.Entries[0].Message,
					"Incorrect log entry message",
				)
			}

			assert.Equal(t, ex.want.statusCode, w.Result().StatusCode, "Incorrect response code")
			b, err := ioutil.ReadAll(w.Result().Body)
			w.Result().Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if len(ex.want.body) > 0 {
				assert.Regexp(t, ex.want.body, string(b), "Incorrect response body")
			} else {
				err = json.Unmarshal(b, &[]images.Image{})
				assert.Nil(t, err, "Response body should be valid json")
			}

			hook.Reset()
		})
	}
}
