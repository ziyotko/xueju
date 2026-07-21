package handler

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestInspectImageUploadUsesActualJPEGFormat(t *testing.T) {
	var data bytes.Buffer
	if err := jpeg.Encode(&data, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	ext, mimeType, err := inspectImageUpload(data.Bytes())
	if err != nil || ext != ".jpg" || mimeType != "image/jpeg" {
		t.Fatalf("inspectImageUpload() = %q, %q, %v", ext, mimeType, err)
	}
}

func TestInspectImageUploadUsesActualPNGFormat(t *testing.T) {
	canvas := image.NewRGBA(image.Rect(0, 0, 8, 8))
	canvas.Set(0, 0, color.RGBA{R: 255, A: 255})
	var data bytes.Buffer
	if err := png.Encode(&data, canvas); err != nil {
		t.Fatal(err)
	}
	ext, mimeType, err := inspectImageUpload(data.Bytes())
	if err != nil || ext != ".png" || mimeType != "image/png" {
		t.Fatalf("inspectImageUpload() = %q, %q, %v", ext, mimeType, err)
	}
}

func TestInspectImageUploadRejectsNonImage(t *testing.T) {
	if _, _, err := inspectImageUpload([]byte("not an image")); err == nil {
		t.Fatal("inspectImageUpload() unexpectedly accepted non-image data")
	}
}
