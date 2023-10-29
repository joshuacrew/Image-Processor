package shared

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

func GenerateJPG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1080, 1080))

	// Create buffers to hold the encoded images
	jpegBuf := &bytes.Buffer{}
	// Encode the image to JPEG
	if err := jpeg.Encode(jpegBuf, img, nil); err != nil {
		t.Fatal(err)
	}

	// Return the encoded JPEG as byte slices
	return jpegBuf.Bytes()
}
