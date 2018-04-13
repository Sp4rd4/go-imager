package images_test

import (
	"errors"

	"github.com/sp4rd4/go-imager/service/images"
)

var examplesDBAddImage = []struct {
	name    string
	input   []*images.Image
	wantErr error
}{
	{
		"Single valid",
		[]*images.Image{&images.Image{Filename: "filename", UserID: 1}},
		nil,
	},
	{
		"Duplicate image",
		[]*images.Image{&images.Image{Filename: "filename", UserID: 1}, &images.Image{Filename: "filename", UserID: 1}},
		images.ErrUniqueIndexConflict("images"),
	},
	{
		"Missing image",
		[]*images.Image{nil},
		errors.New("image required"),
	},
	{
		"Incorrect image",
		[]*images.Image{&images.Image{Filename: "", UserID: 1}},
		errors.New("image filename required"),
	},
}

var examplesDBLoadImages = []struct {
	name    string
	initial []images.Image
	limit   uint64
	offset  uint64
	userID  uint64
	want    []images.Image
	wantErr error
}{
	{
		"Two records",
		[]images.Image{
			images.Image{Filename: "filename1", UserID: 1},
			images.Image{Filename: "filename2", UserID: 1},
			images.Image{Filename: "filename3", UserID: 2},
		},
		0,
		0,
		1,
		[]images.Image{
			images.Image{Filename: "filename1", UserID: 1},
			images.Image{Filename: "filename2", UserID: 1},
		},
		nil,
	},
	{
		"Three records with limit 1",
		[]images.Image{
			images.Image{Filename: "filename1", UserID: 1},
			images.Image{Filename: "filename3", UserID: 2},
			images.Image{Filename: "filename4", UserID: 1},
		},
		1,
		0,
		1,
		[]images.Image{images.Image{Filename: "filename1", UserID: 1}},
		nil,
	},
	{
		"Two records with limit 2 and offset 1",
		[]images.Image{
			images.Image{Filename: "filename1", UserID: 1},
			images.Image{Filename: "filename2", UserID: 1},
			images.Image{Filename: "filename3", UserID: 2},
			images.Image{Filename: "filename4", UserID: 1},
		},
		2,
		1,
		1,
		[]images.Image{
			images.Image{Filename: "filename2", UserID: 1},
			images.Image{Filename: "filename4", UserID: 1},
		},
		nil,
	},
	{
		"No records",
		[]images.Image{
			images.Image{Filename: "filename1", UserID: 3},
			images.Image{Filename: "filename2", UserID: 4},
			images.Image{Filename: "filename3", UserID: 2},
			images.Image{Filename: "filename4", UserID: 5},
		},
		2,
		1,
		1,
		[]images.Image{},
		nil,
	},
}
