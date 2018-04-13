package imgr_test

import (
	"errors"

	"github.com/sp4rd4/go-imager/service/imgr"
)

var examplesDBAddImage = []struct {
	name    string
	input   []*imgr.Image
	wantErr error
}{
	{
		"Single valid",
		[]*imgr.Image{&imgr.Image{Filename: "filename", UserID: 1}},
		nil,
	},
	{
		"Duplicate image",
		[]*imgr.Image{&imgr.Image{Filename: "filename", UserID: 1}, &imgr.Image{Filename: "filename", UserID: 1}},
		imgr.ErrUniqueIndexConflict("images"),
	},
	{
		"Missing image",
		[]*imgr.Image{nil},
		errors.New("image required"),
	},
	{
		"Incorrect image",
		[]*imgr.Image{&imgr.Image{Filename: "", UserID: 1}},
		errors.New("image filename required"),
	},
}

var examplesDBLoadImages = []struct {
	name    string
	initial []imgr.Image
	limit   uint64
	offset  uint64
	userID  uint64
	want    []imgr.Image
	wantErr error
}{
	{
		"Two records",
		[]imgr.Image{
			imgr.Image{Filename: "filename1", UserID: 1},
			imgr.Image{Filename: "filename2", UserID: 1},
			imgr.Image{Filename: "filename3", UserID: 2},
		},
		0,
		0,
		1,
		[]imgr.Image{
			imgr.Image{Filename: "filename1", UserID: 1},
			imgr.Image{Filename: "filename2", UserID: 1},
		},
		nil,
	},
	{
		"Three records with limit 1",
		[]imgr.Image{
			imgr.Image{Filename: "filename1", UserID: 1},
			imgr.Image{Filename: "filename3", UserID: 2},
			imgr.Image{Filename: "filename4", UserID: 1},
		},
		1,
		0,
		1,
		[]imgr.Image{imgr.Image{Filename: "filename1", UserID: 1}},
		nil,
	},
	{
		"Two records with limit 2 and offset 1",
		[]imgr.Image{
			imgr.Image{Filename: "filename1", UserID: 1},
			imgr.Image{Filename: "filename2", UserID: 1},
			imgr.Image{Filename: "filename3", UserID: 2},
			imgr.Image{Filename: "filename4", UserID: 1},
		},
		2,
		1,
		1,
		[]imgr.Image{
			imgr.Image{Filename: "filename2", UserID: 1},
			imgr.Image{Filename: "filename4", UserID: 1},
		},
		nil,
	},
	{
		"No records",
		[]imgr.Image{
			imgr.Image{Filename: "filename1", UserID: 3},
			imgr.Image{Filename: "filename2", UserID: 4},
			imgr.Image{Filename: "filename3", UserID: 2},
			imgr.Image{Filename: "filename4", UserID: 5},
		},
		2,
		1,
		1,
		[]imgr.Image{},
		nil,
	},
}
