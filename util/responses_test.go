package utils_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/sp4rd4/go-imager/util"
	"github.com/stretchr/testify/assert"
)

type badResponseWriter struct {
	http.ResponseWriter
}

func (b badResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("Network failed")
}

func TestJSONResponse(t *testing.T) {
	logger, hook := test.NewNullLogger()
	w := httptest.NewRecorder()
	entry := log.NewEntry(logger)
	t.Run("Able to write response", func(t *testing.T) {
		utils.JSONResponse(w, http.StatusOK, "message", entry)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "Mismatching response content type")
		assert.Equal(t, http.StatusOK, w.Result().StatusCode, "Mismatching response status code")
		b, err := ioutil.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "message", string(b), "Mismatching response body content")
		assert.Equal(t, 0, len(hook.Entries))
		hook.Reset()
	})

	w = httptest.NewRecorder()
	bw := badResponseWriter{w}
	entry = log.NewEntry(logger)
	t.Run("Not able to write response", func(t *testing.T) {
		utils.JSONResponse(bw, http.StatusOK, "message", entry)
		assert.Equal(t, "application/json", bw.Header().Get("Content-Type"), "Mismatching response content type")
		assert.Equal(t, http.StatusOK, w.Result().StatusCode, "Mismatching response status code")
		b, err := ioutil.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "", string(b), "Mismatching response body content")
		if assert.Equal(t, 1, len(hook.Entries)) {
			assert.Equal(t, "Network failed", hook.Entries[0].Message, "Wrong log entry message")
		}
		hook.Reset()
	})
}
