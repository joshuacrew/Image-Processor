package main

import (
	"main/image_put"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(image_put.HandleRequest)
}
