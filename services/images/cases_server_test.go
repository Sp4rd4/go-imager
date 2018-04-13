package images_test

import (
	"net/http"

	"github.com/sp4rd4/go-imager/utils"
)

const (
	fileMissing byte = iota
	fileValid
	fileBroken
	fileNonImage
)

type requestPost struct {
	file    byte
	context map[utils.RequestKey]interface{}
}
type requestList struct {
	limit   uint64
	offset  uint64
	context map[utils.RequestKey]interface{}
}

type want struct {
	body       string
	statusCode int
	logMessage string
}

type stubUser uint64

func (u stubUser) ID() uint64 {
	return uint64(u)
}

var examplesLocalImageServerPostImage = []struct {
	name       string
	staticPath bool
	storage    bool
	requestPost
	want
}{
	{
		name:       "OK",
		staticPath: true,
		storage:    true,
		requestPost: requestPost{
			file:    fileValid,
			context: map[utils.RequestKey]interface{}{utils.RequestUserKey: stubUser(1)},
		},
		want: want{
			statusCode: http.StatusCreated,
		},
	},
	{
		name:       "No user",
		staticPath: true,
		storage:    true,
		requestPost: requestPost{
			file:    fileValid,
			context: map[utils.RequestKey]interface{}{},
		},
		want: want{
			body:       `{"error":"Unauthorized"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "no valid user_id provided",
		},
	},
	{
		name:       "No body",
		staticPath: true,
		storage:    true,
		requestPost: requestPost{
			file:    fileMissing,
			context: map[utils.RequestKey]interface{}{utils.RequestUserKey: stubUser(1)},
		},
		want: want{
			body:       `{"error":"No image is present"}`,
			statusCode: http.StatusUnprocessableEntity,
			logMessage: "request Content-Type isn't multipart/form-data",
		},
	},
	{
		name:       "Not image",
		staticPath: true,
		storage:    true,
		requestPost: requestPost{
			file:    fileNonImage,
			context: map[utils.RequestKey]interface{}{utils.RequestUserKey: stubUser(1)},
		},
		want: want{
			body:       `{"error":"No image is present"}`,
			statusCode: http.StatusUnprocessableEntity,
			logMessage: "content type is incorrect",
		},
	},
	{
		name:       "Invalid form",
		staticPath: true,
		storage:    true,
		requestPost: requestPost{
			file:    fileBroken,
			context: map[utils.RequestKey]interface{}{utils.RequestUserKey: stubUser(1)},
		},
		want: want{
			body:       `{"error":"No image is present"}`,
			statusCode: http.StatusUnprocessableEntity,
			logMessage: "content type is incorrect",
		},
	},
	{
		name:       "Bad static folder",
		staticPath: false,
		storage:    true,
		requestPost: requestPost{
			file:    fileValid,
			context: map[utils.RequestKey]interface{}{utils.RequestUserKey: stubUser(1)},
		},
		want: want{
			body:       `{"error":"Internal server error"}`,
			statusCode: http.StatusInternalServerError,
			logMessage: "no such file or directory",
		},
	},
	{
		name:       "Bad storage",
		staticPath: true,
		storage:    false,
		requestPost: requestPost{
			file:    fileValid,
			context: map[utils.RequestKey]interface{}{utils.RequestUserKey: stubUser(1)},
		},
		want: want{
			body:       `{"error":"Internal server error"}`,
			statusCode: http.StatusInternalServerError,
			logMessage: "storage error",
		},
	},
}

var examplesLocalImageServerListImages = []struct {
	name    string
	storage bool
	requestList
	want
}{
	{
		name:    "OK",
		storage: true,
		requestList: requestList{
			context: map[utils.RequestKey]interface{}{utils.RequestUserKey: stubUser(1)},
		},
		want: want{
			statusCode: http.StatusOK,
		},
	},
	{
		name:    "OK with params",
		storage: true,
		requestList: requestList{
			offset:  1,
			limit:   1,
			context: map[utils.RequestKey]interface{}{utils.RequestUserKey: stubUser(1)},
		},
		want: want{
			statusCode: http.StatusOK,
		},
	},
	{
		name:    "No user",
		storage: true,
		requestList: requestList{
			context: map[utils.RequestKey]interface{}{},
		},
		want: want{
			body:       `{"error":"Unauthorized"}`,
			statusCode: http.StatusUnauthorized,
			logMessage: "no valid user_id provided",
		},
	},
	{
		name:    "Bad storage",
		storage: false,
		requestList: requestList{
			offset:  1,
			limit:   6,
			context: map[utils.RequestKey]interface{}{utils.RequestUserKey: stubUser(1)},
		},
		want: want{
			body:       `{"error":"Internal server error"}`,
			statusCode: http.StatusInternalServerError,
			logMessage: "storage error",
		},
	},
}
