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

var api_gateway_url = "https://suez8r5h95.execute-api.eu-west-2.amazonaws.com/dev/images/"

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

			// Make a POST request to the URL
			resp, err := http.Post(api_gateway_url, "application/json", bytes.NewBuffer(bodyJSON))
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

			// Make a GET request to the URL
			resp, err := http.Get(url.String())
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
