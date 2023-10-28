package image_put

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func generateJPG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1280, 720))

	// Create buffers to hold the encoded images
	jpegBuf := &bytes.Buffer{}
	// Encode the image to JPEG
	if err := jpeg.Encode(jpegBuf, img, nil); err != nil {
		t.Fatal(err)
	}

	// Return the encoded JPEG as byte slices
	return jpegBuf.Bytes()
}

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
			imageData:     generateJPG(t), // Valid JPEG data
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
			_, err := tryConvertToJPEG(tc.imageData)
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

type mockPutObjectAPI func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)

// PutObject implements S3PutObjectAPI.
func (m mockPutObjectAPI) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m(ctx, params, optFns...)
}

func TestUploadImageToS3(t *testing.T) {
	testCases := []struct {
		name            string
		bucketName      string
		imageData       []byte
		objName         string
		expectErr       bool
		s3ResponseError error
	}{
		{
			name:            "SuccessfulUpload",
			bucketName:      "validBucket",
			imageData:       []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00},
			objName:         "image.jpg",
			expectErr:       false,
			s3ResponseError: nil,
		},
		{
			name:            "S3UploadFailure",
			bucketName:      "errorBucket",
			imageData:       []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00},
			objName:         "image.jpg",
			expectErr:       true,
			s3ResponseError: errors.New("S3 upload failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the S3 error if expected
			client := func() S3PutObjectAPI {
				return mockPutObjectAPI(func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					return &s3.PutObjectOutput{}, tc.s3ResponseError
				})
			}

			err := uploadImageToS3(context.TODO(), client(), tc.imageData, tc.objName)
			if (err != nil) != tc.expectErr {
				t.Errorf("Expected error: %v, got: %v", tc.expectErr, err)
			}
		})
	}
}

func TestHandler(t *testing.T) {
	testCases := []struct {
		name            string
		requestBody     any
		expectStatus    int
		expectResponse  string
		s3ResponseError error
	}{
		{
			name:            "ValidImageRequest",
			requestBody:     ImageRequest{generateJPG(t), "image.jpg"},
			expectStatus:    200,
			expectResponse:  `{"message": "Image received, is valid, and has been uploaded to S3."}`,
			s3ResponseError: nil,
		},
		{
			name:           "InvalidRequestBody",
			requestBody:    "Invalid",
			expectStatus:   400,
			expectResponse: `{"message": "Invalid request body"}`,
		},
		{
			name: "InvalidRequestBodyJson",
			requestBody: struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}{
				Key:   "abc",
				Value: "def",
			},
			expectStatus:   400,
			expectResponse: `{"message": "Invalid request body structure"}`,
		},
		{
			name:           "InvalidImage",
			requestBody:    ImageRequest{[]byte{0x01, 0x02, 0x03, 0x04, 0x05}, "image.jpg"},
			expectStatus:   400,
			expectResponse: `{"message": "Invalid image"}`,
		},
		{
			name:            "s3Error",
			requestBody:     ImageRequest{generateJPG(t), "image.jpg"},
			expectStatus:    500,
			expectResponse:  `{"message": "Error uploading image to S3"}`,
			s3ResponseError: errors.New("S3 upload failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal the request body struct to JSON
			bodyJSON, _ := json.Marshal(tc.requestBody)

			request := events.APIGatewayProxyRequest{
				Body: string(bodyJSON),
			}

			client := func() S3PutObjectAPI {
				return mockPutObjectAPI(func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					return &s3.PutObjectOutput{}, tc.s3ResponseError
				})
			}

			s3Client = client()

			response, err := HandleRequest(context.Background(), request)
			if err != nil && tc.s3ResponseError == nil {
				t.Errorf("Handler returned an error: %v", err)
			}

			if response.StatusCode != tc.expectStatus {
				t.Errorf("Expected status code %d, got: %d", tc.expectStatus, response.StatusCode)
			}

			if response.Body != tc.expectResponse {
				t.Errorf("Expected response body: %s, got: %s", tc.expectResponse, response.Body)
			}
		})
	}
}
