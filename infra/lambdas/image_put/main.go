package main

import (
	"image_put/image_put_lambda"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(image_put_lambda.HandleRequest)
}
