package utils_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sp4rd4/go-imager/utils"
	"github.com/stretchr/testify/assert"
)

func TestJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	utils.JSONResponse(w, http.StatusOK, "message")
	assert := assert.New(t)
	assert.Equal("application/json", w.Header().Get("Content-Type"), "Mismatching response content type")
	assert.Equal(http.StatusOK, w.Result().StatusCode, "Mismatching response status code")
	b, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal("message", string(b), "Mismatching response body content")
}
