package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image_put/image_put_lambda"
	"log"
	"net/http"
	"net/url"
	"shared"
	"testing"
)

var api_gateway_url = "https://9v3i8t156j.execute-api.eu-west-2.amazonaws.com/dev/images/"
var auth_token = "eyJraWQiOiJVQmFicGdYN0l6d2hDbmVIelZLQWtEMjFEVlB1TXc1S25VTUtFUEh4TTFBPSIsImFsZyI6IlJTMjU2In0.eyJzdWIiOiIzZGViMjBhYS1mZTczLTQwNjEtODg2ZS0xNzViNWY0M2NhYzEiLCJlbWFpbF92ZXJpZmllZCI6ZmFsc2UsImlzcyI6Imh0dHBzOlwvXC9jb2duaXRvLWlkcC5ldS13ZXN0LTIuYW1hem9uYXdzLmNvbVwvZXUtd2VzdC0yX3RYcjlMdG9KciIsImNvZ25pdG86dXNlcm5hbWUiOiJtZWRpYWZseSIsIm9yaWdpbl9qdGkiOiIwMWRmNmY1ZC1jYWJiLTQ0NGEtYThjYy0wYjdmNjFhNWZmM2IiLCJhdWQiOiIybW85am4ycjU2cDNjNWxmdjgyOW9wcDdmbyIsImV2ZW50X2lkIjoiMzhmZGU1MmUtN2Y3ZC00NWU0LTkzNzctODY3OTk4NjA2ODlkIiwidG9rZW5fdXNlIjoiaWQiLCJhdXRoX3RpbWUiOjE2OTg2OTE4ODMsIm5hbWUiOiJ0ZXN0IiwiZXhwIjoxNjk4Njk1NDgzLCJpYXQiOjE2OTg2OTE4ODMsImp0aSI6Ijg0ZjkzMDk3LThiNTYtNDM3OS04YTYxLTlmODBjYjdkNDIzZiIsImVtYWlsIjoidGVzdEB0ZXN0LmNvbSJ9.F7-BUc25TvUFFTv4aAirsuyX45LxKyWop7sFqrmtfyi0kHzatzhihM8pcWlLmcJNuPmzQgx8u60XDBHKe5zeyHi4IKD1sMMAjESmfLX3lQ6Fm1uSLzVqRQDUxDi_BZRyK-stCOWH26uAnyqVQW9shPswIUv8LubjGLa4mYYXSYbViUy5umXldPXo8b5U4Ex0n_n9EhaSGYmJ7juNHOJEHSiCepIOZMFyU3vwbz37N9JAYLGTXDJiYGjFQqR7FVuSJLldzc9TsjGM3bzagGDdnLgoU29zmt7LgFAu0xPUnQJaOMbSgQydJmLEnLDi--1cvC9-XVoXCYMg4vfCBsXn-w"

func TestPostImageHandler(t *testing.T) {
	testCases := []struct {
		name             string
		request          any
		expectedStatus   int
		expectedResponse string
	}{
		{
			name: "Valid JPEG Image",
			request: image_put_lambda.ImageRequest{
				ImageData: shared.GenerateJPG(t),
				ImageName: "image.jpg",
			},
			expectedStatus:   200,
			expectedResponse: `{"message": "Image received, is valid, and has been uploaded to S3."}`,
		},
		{
			name: "Invalid Image Format",
			request: image_put_lambda.ImageRequest{
				ImageData: []byte("invalid image data"), // Invalid format
				ImageName: "invalid.png",
			},
			expectedStatus:   400,
			expectedResponse: `{"message": "Invalid image"}`,
		},
		{
			name:             "Invalid Request Format",
			request:          "hello",
			expectedStatus:   400,
			expectedResponse: `{"message": "Invalid request body"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request body
			bodyJSON, _ := json.Marshal(tc.request)

			// Create the request with the Authorization header
			req, err := http.NewRequest("POST", api_gateway_url, bytes.NewBuffer(bodyJSON))
			if err != nil {
				t.Fatalf("Failed to create the request: %v", err)
			}
			req.Header.Add("Authorization", auth_token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to make the POST request: %v", err)
			}
			defer resp.Body.Close()

			// Check the status code
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status code %d, got: %d", tc.expectedStatus, resp.StatusCode)
			}

			// Read the response body
			buffer := new(bytes.Buffer)
			_, err = buffer.ReadFrom(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read the response body: %v", err)
			}

			// Check the response message
			if buffer.String() != tc.expectedResponse {
				t.Errorf("Expected response: %s, got: %s", tc.expectedResponse, buffer.String())
			}
		})
	}
}

func TestGetImageHandler(t *testing.T) {
	rotatedResponse, _ := shared.RotateAndResize(shared.GenerateJPG(t))

	testCases := []struct {
		name             string
		queryParams      url.Values
		expectedStatus   int
		expectedResponse string
	}{
		{
			name:             "Valid JPEG Image",
			queryParams:      url.Values{"name": {"image.jpg"}},
			expectedStatus:   200,
			expectedResponse: base64.StdEncoding.EncodeToString((shared.GenerateJPG(t))),
		},
		{
			name:             "Valid JPEG Image with rotate",
			queryParams:      url.Values{"name": {"image.jpg"}, "rotate": {"true"}},
			expectedStatus:   200,
			expectedResponse: base64.StdEncoding.EncodeToString(rotatedResponse),
		},
		{
			name:             "Image Not Found",
			queryParams:      url.Values{"name": {"image.jgp"}},
			expectedStatus:   404,
			expectedResponse: `{"message": "Image with name image.jgp not found in S3"}`,
		},
		{
			name:             "Invalid Request Format",
			queryParams:      url.Values{"hello": {"image.jpg"}},
			expectedStatus:   400,
			expectedResponse: `{"message": "Missing 'name' parameter in the URL path"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url, err := url.Parse(api_gateway_url)
			if err != nil {
				log.Fatal(err)
			}
			url.RawQuery = tc.queryParams.Encode()

			req, err := http.NewRequest("GET", url.String(), nil)
			if err != nil {
				t.Fatalf("Failed to create the request: %v", err)
			}
			req.Header.Add("Authorization", auth_token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to make the GET request: %v", err)
			}
			defer resp.Body.Close()

			// Check the status code
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status code %d, got: %d", tc.expectedStatus, resp.StatusCode)
			}

			// Read the response body
			buffer := new(bytes.Buffer)
			_, err = buffer.ReadFrom(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read the response body: %v", err)
			}

			// Check the response message
			if buffer.String() != tc.expectedResponse {
				t.Errorf("Expected response: %s, got: %s", tc.expectedResponse, buffer.String())
			}
		})
	}
}
