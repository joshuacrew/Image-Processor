package main

import (
	"image_get/image_get_lambda"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(image_get_lambda.HandleRequest)
}
