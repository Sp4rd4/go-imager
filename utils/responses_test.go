package utils_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sp4rd4/go-imager/utils"
	"github.com/stretchr/testify/assert"
)

func TestJsonResponse(t *testing.T) {
	w := httptest.NewRecorder()
	utils.JsonResponse(w, http.StatusOK, "message")
	assertJSONResponse(t, w, "application/json", "message", http.StatusOK)
}

func assertJSONResponse(t *testing.T, rec *httptest.ResponseRecorder, contentType, body string, status int) {
	assert := assert.New(t)
	assert.Equal(rec.Header().Get("Content-Type"), contentType)
	assert.Equal(rec.Result().StatusCode, status)
	b, err := ioutil.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(string(b), body)
}
