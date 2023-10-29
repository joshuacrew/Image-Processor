package shared

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func generatePNG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1280, 720))

	// Create buffers to hold the encoded images
	jpegBuf := &bytes.Buffer{}
	// Encode the image to PNG
	if err := png.Encode(jpegBuf, img); err != nil {
		t.Fatal(err)
	}

	// Return the encoded PNG as byte slices
	return jpegBuf.Bytes()
}

func TestTryConvertToJPEG(t *testing.T) {
	testCases := []struct {
		name          string
		imageData     []byte
		expectSuccess bool
	}{
		{
			name:          "ValidJPEGImage",
			imageData:     GenerateJPG(t), // Valid JPEG data
			expectSuccess: true,
		},
		{
			name:          "ValidPNGImage",
			imageData:     generatePNG(t), // Valid PNG data
			expectSuccess: true,
		},
		{
			name:          "InvalidImage",
			imageData:     []byte{0x01, 0x02, 0x03, 0x04, 0x05}, // Invalid image data
			expectSuccess: false,
		},
		{
			name:          "InvalidFormatImage",
			imageData:     []byte("This is not an image"), // Non-image data
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := TryConvertToJPEG(tc.imageData)
			if tc.expectSuccess {
				if err != nil {
					t.Errorf("Expected successful conversion but got an error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected an error but conversion was successful")
				}
			}
		})
	}
}
