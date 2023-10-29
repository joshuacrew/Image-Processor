package image_get_lambda

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"shared"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// RequestBody is the structure of the request body.
type RequestBody struct {
	Name string `json:"name"`
}

var s3Client shared.S3ObjectAPI

func init() {
	var err error
	s3Client, err = shared.NewS3Client()
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}
}

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get the 'name' parameter from the URL path
	name := request.QueryStringParameters["name"]
	if name == "" {
		log.Println("Missing 'name' parameter in the URL path")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "Missing 'name' parameter in the URL path"}`,
		}, nil
	}

	output, err := getImageFromS3(context.TODO(), s3Client, name)
	if err != nil {
		// Check if the error represents a "Not Found" condition
		var responseError *awshttp.ResponseError
		if errors.As(err, &responseError) && responseError.ResponseError.HTTPStatusCode() == http.StatusNotFound {
			return events.APIGatewayProxyResponse{
				StatusCode: 404,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       fmt.Sprintf(`{"message": "Image with name %s not found in S3"}`, name),
			}, nil
		}
		// Handle other errors
		log.Printf("Error retrieving image from S3: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "Failed to retrieve object from S3"}`,
		}, err
	}

	// Read the object's content into a []byte
	body, readErr := io.ReadAll(output.Body)
	if readErr != nil {
		log.Printf("Error reading image content: %v", readErr)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "Failed to read object content"}`,
		}, readErr
	}

	// Check if the 'rotate' query parameter is present and set to "true"
	rotateParam, ok := request.QueryStringParameters["rotate"]
	if ok && rotateParam == "true" {
		log.Println("Rotating image by 180 degrees")
		rotatedImageBytes, err := shared.RotateAndResize(body)
		if err != nil {
			log.Printf("Error rotating and resizing: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"message": "Failed to rotate and resize"}`,
			}, err
		}

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "image/jpeg"},
			Body:       base64.StdEncoding.EncodeToString(rotatedImageBytes),
		}, nil
	}

	// Build the response
	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "image/jpeg"},
		Body:       base64.StdEncoding.EncodeToString(body),
	}

	return response, nil
}

// Gets the image from Amazon S3
func getImageFromS3(ctx context.Context, s3Client shared.S3ObjectAPI, name string) (*s3.GetObjectOutput, error) {
	bucketName := os.Getenv("S3_BUCKET_NAME")
	log.Printf("bucketName: %s", bucketName)

	output, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(name),
	})

	return output, err
}
