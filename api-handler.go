package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
)

func listLightsailInstances(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		http.Error(w, "Please provide a 'region' query parameter", http.StatusBadRequest)
		return
	}

	// Replace these with your AWS access key and secret key
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	staticProvider := credentials.NewStaticCredentialsProvider(
		accessKey,
		secretKey,
		"",
	)

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(staticProvider),
		config.WithRegion(region),
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading AWS configuration: %v", err), http.StatusInternalServerError)
		return
	}

	client := lightsail.NewFromConfig(cfg)

	params := &lightsail.GetInstancesInput{}
	resp, err := client.GetInstances(context.TODO(), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing instances: %v", err), http.StatusInternalServerError)
		return
	}

	instances := []string{}
	for _, instance := range resp.Instances {
		instanceStatus := fmt.Sprintf("Name: %s, PublicIP: %s, State: %s",
			*instance.Name, *instance.PublicIpAddress, *instance.State.Name)
		instances = append(instances, instanceStatus)
	}

	responseJSON, err := json.Marshal(instances)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

func resetLightsailInstance(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	instanceName := r.URL.Query().Get("name")

	if region == "" || instanceName == "" {
		http.Error(w, "Please provide 'region' and 'name' query parameters", http.StatusBadRequest)
		return
	}

	// Replace these with your AWS access key and secret key
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	staticProvider := credentials.NewStaticCredentialsProvider(
		accessKey,
		secretKey,
		"",
	)

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(staticProvider),
		config.WithRegion(region),
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading AWS configuration: %v", err), http.StatusInternalServerError)
		return
	}

	client := lightsail.NewFromConfig(cfg)

	// Use the "name" query parameter to identify the instance
	params := &lightsail.RebootInstanceInput{
		InstanceName: aws.String(instanceName),
	}
	resp, err := client.RebootInstance(context.TODO(), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting instance: %v", err), http.StatusInternalServerError)
		return
	}

	// Add the reset logic here
	// You can use the "name" query parameter to reset the specific instance

	responseJSON, err := json.Marshal(resp.Operations)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)

}

func main() {
	http.HandleFunc("/api/instances", listLightsailInstances)
	http.HandleFunc("/api/instance", resetLightsailInstance)

	fmt.Println("Server is running on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
