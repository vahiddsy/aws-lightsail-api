package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
)

func main() {
	// Replace these with your AWS access key and secret key
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	// Replace "us-east-1" with your desired AWS region
	selectedRegion := "us-west-1"

	// Regions a list of values
	values := []string{"us-east-1", "us-east-2", "us-west-2", "eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2", "ap-south-1", "ca-central-1", "eu-north-1"}

	// Display options to the user
	fmt.Println("Select an Region:")
	for i, option := range values {
		fmt.Printf("%d. %s\n", i+1, option)
	}
	// Read user input
	var userInput int
	fmt.Print("Enter the number of your choice: ")
	_, err := fmt.Scanf("%d", &userInput)
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Check the user's choice and select the corresponding value
	if userInput >= 1 && userInput <= len(values) {
		selectedRegion = values[userInput-1]
	}

	staticProvider := credentials.NewStaticCredentialsProvider(
		accessKey,
		secretKey,
		"",
	)

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(staticProvider),
		config.WithRegion(selectedRegion),
	)

	// cfg, err := config.LoadDefaultConfig(context.TODO(),
	// 	config.WithCredentialsProvider{
	// 		StaticCredentialsProvider{
	// 			Value: config.Credentials{
	// 				AccessKeyID:     accessKey,
	// 				SecretAccessKey: secretKey,
	// 			},
	// 		},
	// 	},
	// )
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		return
	}

	// Create a Lightsail client
	client := lightsail.NewFromConfig(cfg)

	// List instances
	params := &lightsail.GetInstancesInput{}
	resp, err := client.GetInstances(context.TODO(), params)
	if err != nil {
		fmt.Println("Error listing instances:", err)
		return
	}

	// Print instance information in the selected region
	fmt.Printf("Lightsail Instances in %s:\n", selectedRegion)
	for _, instance := range resp.Instances {
		fmt.Printf("Name: %s\n", *instance.Name)
		fmt.Printf("ID: %s\n", *instance.Arn)
		fmt.Printf("PublicIP: %s\n", *instance.PublicIpAddress)
		fmt.Printf("State: %s\n", *instance.State.Name)

		fmt.Println("------------")
	}
}
