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

var AUTH_KEY = "eyJraWQiOiJVQmFicGdYN0l6d2hDbmVIelZLQWtEMjFEVlB1TXc1S25VTUtFUEh4TTFBPSIsImFsZyI6IlJTMjU2In0.eyJzdWIiOiIzZGViMjBhYS1mZTczLTQwNjEtODg2ZS0xNzViNWY0M2NhYzEiLCJlbWFpbF92ZXJpZmllZCI6ZmFsc2UsImlzcyI6Imh0dHBzOlwvXC9jb2duaXRvLWlkcC5ldS13ZXN0LTIuYW1hem9uYXdzLmNvbVwvZXUtd2VzdC0yX3RYcjlMdG9KciIsImNvZ25pdG86dXNlcm5hbWUiOiJtZWRpYWZseSIsIm9yaWdpbl9qdGkiOiIwMWRmNmY1ZC1jYWJiLTQ0NGEtYThjYy0wYjdmNjFhNWZmM2IiLCJhdWQiOiIybW85am4ycjU2cDNjNWxmdjgyOW9wcDdmbyIsImV2ZW50X2lkIjoiMzhmZGU1MmUtN2Y3ZC00NWU0LTkzNzctODY3OTk4NjA2ODlkIiwidG9rZW5fdXNlIjoiaWQiLCJhdXRoX3RpbWUiOjE2OTg2OTE4ODMsIm5hbWUiOiJ0ZXN0IiwiZXhwIjoxNjk4Njk1NDgzLCJpYXQiOjE2OTg2OTE4ODMsImp0aSI6Ijg0ZjkzMDk3LThiNTYtNDM3OS04YTYxLTlmODBjYjdkNDIzZiIsImVtYWlsIjoidGVzdEB0ZXN0LmNvbSJ9.F7-BUc25TvUFFTv4aAirsuyX45LxKyWop7sFqrmtfyi0kHzatzhihM8pcWlLmcJNuPmzQgx8u60XDBHKe5zeyHi4IKD1sMMAjESmfLX3lQ6Fm1uSLzVqRQDUxDi_BZRyK-stCOWH26uAnyqVQW9shPswIUv8LubjGLa4mYYXSYbViUy5umXldPXo8b5U4Ex0n_n9EhaSGYmJ7juNHOJEHSiCepIOZMFyU3vwbz37N9JAYLGTXDJiYGjFQqR7FVuSJLldzc9TsjGM3bzagGDdnLgoU29zmt7LgFAu0xPUnQJaOMbSgQydJmLEnLDi--1cvC9-XVoXCYMg4vfCBsXn-w"

func TestHandler(t *testing.T) {
	testCases := []struct {
		name            string
		requestBody     any
		expectStatus    int
		expectResponse  string
		s3ResponseError error
		authKey         string
	}{
		{
			name:            "ValidImageRequest",
			requestBody:     ImageRequest{shared.GenerateJPG(t), "image.jpg"},
			expectStatus:    200,
			expectResponse:  `{"message": "Image received, is valid, and has been uploaded to S3."}`,
			s3ResponseError: nil,
			authKey:         AUTH_KEY,
		},
		{
			name:           "InvalidRequestBody",
			requestBody:    "Invalid",
			expectStatus:   400,
			expectResponse: `{"message": "Invalid request body"}`,
			authKey:        AUTH_KEY,
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
			authKey:        AUTH_KEY,
		},
		{
			name:           "InvalidImage",
			requestBody:    ImageRequest{[]byte{0x01, 0x02, 0x03, 0x04, 0x05}, "image.jpg"},
			expectStatus:   400,
			expectResponse: `{"message": "Invalid image"}`,
			authKey:        AUTH_KEY,
		},
		{
			name:            "s3Error",
			requestBody:     ImageRequest{shared.GenerateJPG(t), "image.jpg"},
			expectStatus:    500,
			expectResponse:  `{"message": "Error uploading image to S3"}`,
			s3ResponseError: errors.New("S3 upload failed"),
			authKey:         AUTH_KEY,
		},
		{
			name:           "Failed to authorize",
			requestBody:    ImageRequest{shared.GenerateJPG(t), "image.jpg"},
			expectStatus:   402,
			expectResponse: `{"message": "Unauthorized"}`,
			authKey:        "abcdef",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal the request body struct to JSON
			bodyJSON, _ := json.Marshal(tc.requestBody)

			request := events.APIGatewayProxyRequest{
				Body: string(bodyJSON),
				Headers: map[string]string{
					"Authorization": tc.authKey,
				},
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
