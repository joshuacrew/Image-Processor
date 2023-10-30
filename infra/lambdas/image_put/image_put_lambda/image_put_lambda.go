package image_put_lambda

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"shared"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ImageRequest struct {
	ImageData []byte `json:"imageData"`
	ImageName string `json:"imageName"`
}

var s3Client shared.S3ObjectAPI

func init() {
	var err error
	s3Client, err = shared.NewS3Client()
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}
}

var jwksURL = "https://cognito-idp.eu-west-2.amazonaws.com/eu-west-2_tXr9LtoJr/.well-known/jwks.json"

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Check if the token is valid
	err := shared.HandleAuth(request.Headers, jwksURL)
	if err != nil {
		// Handle authentication error
		log.Printf("Authentication error: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 401, // Unauthorized
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "Unauthorized"}`,
		}, nil
	}

	// Unmarshal the request body into an ImageRequest struct
	var imageRequest ImageRequest
	if err := json.Unmarshal([]byte(request.Body), &imageRequest); err != nil {
		log.Printf("Error unmarshaling request body: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "Invalid request body"}`,
		}, nil
	}

	// Check that all fields are non-empty
	if len(imageRequest.ImageData) == 0 || imageRequest.ImageName == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "Invalid request body structure"}`,
		}, nil
	}

	// Check if image can be converted to jpeg
	jpeg, err := shared.TryConvertToJPEG(imageRequest.ImageData)
	if err != nil {
		log.Printf("Error converting image to JPEG: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "Invalid image"}`,
		}, nil
	}

	if err := uploadImageToS3(context.TODO(), s3Client, jpeg, imageRequest.ImageName); err != nil {
		log.Printf("Error uploading image to S3: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "Error uploading image to S3"}`,
		}, err
	}

	log.Println("Image successfully uploaded to S3.")

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"message": "Image received, is valid, and has been uploaded to S3."}`,
	}, nil
}

// Upload the image to Amazon S3
func uploadImageToS3(ctx context.Context, s3Client shared.S3ObjectAPI, imageData []byte, name string) error {
	bucketName := os.Getenv("S3_BUCKET_NAME")
	log.Printf("bucketName: %s", bucketName)

	_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(name),
		Body:        bytes.NewReader(imageData),
		ContentType: aws.String("image/jpeg"),
	})

	return err
}
