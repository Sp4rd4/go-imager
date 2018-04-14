package imgr_test

import (
	"errors"

	"github.com/sp4rd4/go-imager/service/imgr"
)

var examplesDBCreateImage = []struct {
	name    string
	input   []*imgr.Image
	wantErr error
}{
	{
		name:    "Single valid",
		input:   []*imgr.Image{{Filename: "filename", UserID: 1}},
		wantErr: nil,
	},
	{
		name:    "Duplicate image",
		input:   []*imgr.Image{{Filename: "filename", UserID: 1}, {Filename: "filename", UserID: 1}},
		wantErr: imgr.ErrUniqueIndexConflict("images"),
	},
	{
		name:    "Missing image",
		input:   []*imgr.Image{nil},
		wantErr: errors.New("image required"),
	},
	{
		name:    "Incorrect image",
		input:   []*imgr.Image{{Filename: "", UserID: 1}},
		wantErr: errors.New("image filename required"),
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
		name: "Two records",
		initial: []imgr.Image{
			{Filename: "filename1", UserID: 1},
			{Filename: "filename2", UserID: 1},
			{Filename: "filename3", UserID: 2},
		},
		limit:  0,
		offset: 0,
		userID: 1,
		want: []imgr.Image{
			{Filename: "filename1", UserID: 1},
			{Filename: "filename2", UserID: 1},
		},
		wantErr: nil,
	},
	{
		name: "Three records with limit 1",
		initial: []imgr.Image{
			{Filename: "filename1", UserID: 1},
			{Filename: "filename3", UserID: 2},
			{Filename: "filename4", UserID: 1},
		},
		limit:   1,
		offset:  0,
		userID:  1,
		want:    []imgr.Image{{Filename: "filename1", UserID: 1}},
		wantErr: nil,
	},
	{
		name: "Two records with limit 2 and offset 1",
		initial: []imgr.Image{
			{Filename: "filename1", UserID: 1},
			{Filename: "filename2", UserID: 1},
			{Filename: "filename3", UserID: 2},
			{Filename: "filename4", UserID: 1},
		},
		limit:  2,
		offset: 1,
		userID: 1,
		want: []imgr.Image{
			{Filename: "filename2", UserID: 1},
			{Filename: "filename4", UserID: 1},
		},
		wantErr: nil,
	},
	{
		name: "No records",
		initial: []imgr.Image{
			{Filename: "filename1", UserID: 3},
			{Filename: "filename2", UserID: 4},
			{Filename: "filename3", UserID: 2},
			{Filename: "filename4", UserID: 5},
		},
		limit:   2,
		offset:  1,
		userID:  1,
		want:    []imgr.Image{},
		wantErr: nil,
	},
}
