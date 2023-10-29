package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"net/url"
	"testing"
)

type ImageRequest struct {
	ImageData []byte `json:"imageData"`
	ImageName string `json:"imageName"`
}

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

func TestPostImageHandler(t *testing.T) {
	testCases := []struct {
		name             string
		request          any
		expectedStatus   int
		expectedResponse string
	}{
		{
			name: "Valid JPEG Image",
			request: ImageRequest{
				ImageData: generateJPG(t),
				ImageName: "image.jpg",
			},
			expectedStatus:   200,
			expectedResponse: `{"message": "Image received, is valid, and has been uploaded to S3."}`,
		},
		{
			name: "Invalid Image Format",
			request: ImageRequest{
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
			url := "https://ca2y5dnlsm2roqr43ll4k47wui0nlyxv.lambda-url.eu-west-2.on.aws/"

			// Create a request body
			bodyJSON, _ := json.Marshal(tc.request)

			// Make a POST request to the URL
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyJSON))
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
			expectedResponse: base64.StdEncoding.EncodeToString((generateJPG(t))),
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
			url, err := url.Parse("https://2y5qt672eeqa4snba4p7kdaa6m0xkmsw.lambda-url.eu-west-2.on.aws/")
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
