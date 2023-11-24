package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

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
		var instanceStatus string
		if *instance.State.Name != "stopped" {
			instanceStatus = fmt.Sprintf("Name: %s, State: %s, PublicIP: %s",
				*instance.Name, *instance.State.Name, *instance.PublicIpAddress)
		} else {
			instanceStatus = fmt.Sprintf("Name: %s, State: %s",
				*instance.Name, *instance.State.Name)
		}
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
	// Parse region query parameter
	region := r.URL.Query().Get("region")
	// Parse name query parameter
	instanceName := r.URL.Query().Get("name")
	// Parse timestamp query parameter
	timestampParam := r.URL.Query().Get("secret")

	if region == "" || instanceName == "" || timestampParam == "" {
		http.Error(w, "Please provide 'region' and 'name' and 'secret' query parameters", http.StatusBadRequest)
		return
	}

	// Convert timestamp to integer
	timestamp, err := strconv.ParseInt(timestampParam, 10, 64)
	if err != nil {
		http.Error(w, "Invalid timestamp format", http.StatusBadRequest)
		return
	}

	// Get current time
	now := time.Now().Unix()

	// Check if the timestamp is within a 2-minute difference from now
	if now-timestamp > 120 || timestamp-now > 120 {
		http.Error(w, "Invalid request: Timestamp is more than 2 minutes difference from current time", http.StatusBadRequest)
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

func listLightsailRegions(w http.ResponseWriter, r *http.Request) {

	values := []string{"us-east-1", "us-east-2", "us-west-2", "eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2", "ap-south-1", "ca-central-1", "eu-north-1"}

	responseJSON, err := json.Marshal(values)
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
	http.HandleFunc("/api/regions", listLightsailRegions)

	fmt.Println("Server is running on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
