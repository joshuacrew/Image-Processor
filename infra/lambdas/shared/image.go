package shared

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"log"

	"github.com/disintegration/imaging"
)

// Rotate the image by 180 degrees and resize
func RotateAndResize(body []byte) ([]byte, error) {
	img, err := imaging.Decode(bytes.NewReader(body))
	if err != nil {
		log.Printf("Error decoding image: %v", err)
		return nil, errors.New("error decoding image")
	}

	img = imaging.Rotate180(img)
	img = imaging.Resize(img, 1280, 720, imaging.ResampleFilter{})

	var rotatedImageBuffer bytes.Buffer
	if err := imaging.Encode(&rotatedImageBuffer, img, imaging.JPEG); err != nil {
		log.Printf("Error encoding rotated image: %v", err)
		return nil, errors.New("error encoding rotated image")
	}

	return rotatedImageBuffer.Bytes(), nil
}

// Try to convert the image data to JPEG format
func TryConvertToJPEG(data []byte) ([]byte, error) {
	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Create a buffer to hold the JPEG-encoded image
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, nil)
	if err != nil {
		return nil, err
	}

	// Return the JPEG-encoded data
	return buf.Bytes(), nil
}
