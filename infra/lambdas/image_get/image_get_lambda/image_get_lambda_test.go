package image_get_lambda

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"shared"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type mockGetObjectAPI func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)

// PutObject implements shared.S3ObjectAPI.
func (mockGetObjectAPI) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	panic("unimplemented")
}

func (m mockGetObjectAPI) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m(ctx, params, optFns...)
}

func TestGetFromS3(t *testing.T) {
	testCases := []struct {
		name          string
		expectedError error
		expectErr     bool
	}{
		{
			name:          "Object Found",
			expectedError: nil,
			expectErr:     false,
		},
		{
			name:          "Object Not Found",
			expectedError: errors.New("object not found"),
			expectErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the S3 error if expected
			client := func() shared.S3ObjectAPI {
				return mockGetObjectAPI(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					return nil, tc.expectedError
				})
			}

			_, err := getImageFromS3(context.TODO(), client(), "image.jpg")
			if (err != nil) != tc.expectErr {
				t.Errorf("Expected error: %v, got: %v", tc.expectErr, err)
			}
		})
	}
}

var AUTH_KEY = "eyJraWQiOiJVQmFicGdYN0l6d2hDbmVIelZLQWtEMjFEVlB1TXc1S25VTUtFUEh4TTFBPSIsImFsZyI6IlJTMjU2In0.eyJzdWIiOiIzZGViMjBhYS1mZTczLTQwNjEtODg2ZS0xNzViNWY0M2NhYzEiLCJlbWFpbF92ZXJpZmllZCI6ZmFsc2UsImlzcyI6Imh0dHBzOlwvXC9jb2duaXRvLWlkcC5ldS13ZXN0LTIuYW1hem9uYXdzLmNvbVwvZXUtd2VzdC0yX3RYcjlMdG9KciIsImNvZ25pdG86dXNlcm5hbWUiOiJtZWRpYWZseSIsIm9yaWdpbl9qdGkiOiIwMWRmNmY1ZC1jYWJiLTQ0NGEtYThjYy0wYjdmNjFhNWZmM2IiLCJhdWQiOiIybW85am4ycjU2cDNjNWxmdjgyOW9wcDdmbyIsImV2ZW50X2lkIjoiMzhmZGU1MmUtN2Y3ZC00NWU0LTkzNzctODY3OTk4NjA2ODlkIiwidG9rZW5fdXNlIjoiaWQiLCJhdXRoX3RpbWUiOjE2OTg2OTE4ODMsIm5hbWUiOiJ0ZXN0IiwiZXhwIjoxNjk4Njk1NDgzLCJpYXQiOjE2OTg2OTE4ODMsImp0aSI6Ijg0ZjkzMDk3LThiNTYtNDM3OS04YTYxLTlmODBjYjdkNDIzZiIsImVtYWlsIjoidGVzdEB0ZXN0LmNvbSJ9.F7-BUc25TvUFFTv4aAirsuyX45LxKyWop7sFqrmtfyi0kHzatzhihM8pcWlLmcJNuPmzQgx8u60XDBHKe5zeyHi4IKD1sMMAjESmfLX3lQ6Fm1uSLzVqRQDUxDi_BZRyK-stCOWH26uAnyqVQW9shPswIUv8LubjGLa4mYYXSYbViUy5umXldPXo8b5U4Ex0n_n9EhaSGYmJ7juNHOJEHSiCepIOZMFyU3vwbz37N9JAYLGTXDJiYGjFQqR7FVuSJLldzc9TsjGM3bzagGDdnLgoU29zmt7LgFAu0xPUnQJaOMbSgQydJmLEnLDi--1cvC9-XVoXCYMg4vfCBsXn-w"

func TestHandleRequest(t *testing.T) {
	rotatedResponse, _ := shared.RotateAndResize(shared.GenerateJPG(t))

	tests := []struct {
		name            string
		pathParams      map[string]string
		expectStatus    int
		expectResponse  string
		s3Response      []byte
		s3ResponseError error
		expectError     bool
		authKey         string
	}{
		{
			name:           "Successful request",
			pathParams:     map[string]string{"name": "example.jpg"},
			expectStatus:   200,
			s3Response:     []byte("fake image content"),
			expectResponse: base64.StdEncoding.EncodeToString([]byte("fake image content")),
			authKey:        AUTH_KEY,
		},
		{
			name:           "Successful request with rotate",
			pathParams:     map[string]string{"name": "example.jpg", "rotate": "true"},
			expectStatus:   200,
			s3Response:     shared.GenerateJPG(t),
			expectResponse: base64.StdEncoding.EncodeToString(rotatedResponse),
			authKey:        AUTH_KEY,
		},
		{
			name:           "Unuccessful request with rotate",
			pathParams:     map[string]string{"name": "example.jpg", "rotate": "true"},
			expectStatus:   500,
			s3Response:     []byte("fake image content"),
			expectResponse: `{"message": "Failed to rotate and resize"}`,
			expectError:    true,
			authKey:        AUTH_KEY,
		},
		{
			name:           "Missing 'name' parameter in path",
			pathParams:     map[string]string{"invalid": "invalid"},
			expectStatus:   400,
			expectResponse: `{"message": "Missing 'name' parameter in the URL path"}`,
			authKey:        AUTH_KEY,
		},
		{
			name:            "Failed to retrieve object from S3",
			pathParams:      map[string]string{"name": "example.jpg"},
			expectStatus:    500,
			expectResponse:  `{"message": "Failed to retrieve object from S3"}`,
			s3ResponseError: errors.New("ERROR"),
			expectError:     true,
			authKey:         AUTH_KEY,
		},
		{
			name:           "Failed to authorize",
			pathParams:     map[string]string{"name": "example.jpg"},
			expectStatus:   401,
			expectResponse: `{"message": "Unauthorized"}`,
			expectError:    true,
			authKey:        "abcdef",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := func() shared.S3ObjectAPI {
				return mockGetObjectAPI(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader(tc.s3Response)),
					}, tc.s3ResponseError
				})
			}

			s3Client = client()

			request := events.APIGatewayProxyRequest{
				QueryStringParameters: tc.pathParams,
				Headers: map[string]string{
					"Authorization": tc.authKey,
				},
			}

			response, err := HandleRequest(context.Background(), request)
			if err != nil && !tc.expectError {
				t.Errorf("Handler returned an error: %v", err)
			}

			if response.StatusCode != tc.expectStatus {
				t.Errorf("Expected status code %d, got: %d", tc.expectStatus, response.StatusCode)
			}

			if response.Body != tc.expectResponse {
				t.Errorf("Expected response body: %s, got: %s", tc.expectResponse, response.Body)
			}
			if response.StatusCode != tc.expectStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectStatus, response.StatusCode)
			}
		})
	}
}
