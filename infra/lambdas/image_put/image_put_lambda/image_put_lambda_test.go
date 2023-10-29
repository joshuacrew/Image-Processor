package image_put_lambda

import (
	"context"
	"encoding/json"
	"errors"
	"shared"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type mockPutObjectAPI func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)

// GetObject implements shared.S3ObjectAPI.
func (mockPutObjectAPI) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	panic("unimplemented")
}

// PutObject implements S3ObjectAPI.
func (m mockPutObjectAPI) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m(ctx, params, optFns...)
}

func TestUploadImageToS3(t *testing.T) {
	testCases := []struct {
		name            string
		imageData       []byte
		expectErr       bool
		s3ResponseError error
	}{
		{
			name:            "SuccessfulUpload",
			imageData:       []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00},
			expectErr:       false,
			s3ResponseError: nil,
		},
		{
			name:            "S3UploadFailure",
			imageData:       []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00},
			expectErr:       true,
			s3ResponseError: errors.New("S3 upload failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the S3 error if expected
			client := func() shared.S3ObjectAPI {
				return mockPutObjectAPI(func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					return &s3.PutObjectOutput{}, tc.s3ResponseError
				})
			}

			err := uploadImageToS3(context.TODO(), client(), tc.imageData, "image.jpg")
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
			requestBody:     ImageRequest{shared.GenerateJPG(t), "image.jpg"},
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
			requestBody:     ImageRequest{shared.GenerateJPG(t), "image.jpg"},
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

			client := func() shared.S3ObjectAPI {
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
