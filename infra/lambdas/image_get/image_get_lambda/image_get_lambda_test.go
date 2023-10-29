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

func TestHandleRequest(t *testing.T) {
	tests := []struct {
		name            string
		pathParams      map[string]string
		expectStatus    int
		expectResponse  string
		s3ResponseError error
	}{
		{
			name:            "Successful request",
			pathParams:      map[string]string{"name": "example.jpg"},
			expectStatus:    200,
			expectResponse:  base64.StdEncoding.EncodeToString([]byte("fake image content")),
			s3ResponseError: nil,
		},
		{
			name:            "Missing 'name' parameter in path",
			pathParams:      map[string]string{"invalid": "invalid"},
			expectStatus:    400,
			expectResponse:  `{"message": "Missing 'name' parameter in the URL path"}`,
			s3ResponseError: nil,
		},
		{
			name:            "Failed to retrieve object from S3",
			pathParams:      map[string]string{"name": "example.jpg"},
			expectStatus:    500,
			expectResponse:  `{"message": "Failed to retrieve object from S3"}`,
			s3ResponseError: errors.New("The specified key does not exist."),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := func() shared.S3ObjectAPI {
				return mockGetObjectAPI(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader([]byte("fake image content"))),
					}, tc.s3ResponseError
				})
			}

			s3Client = client()

			request := events.APIGatewayProxyRequest{
				QueryStringParameters: tc.pathParams,
			}

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
			if response.StatusCode != tc.expectStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectStatus, response.StatusCode)
			}
		})
	}
}
