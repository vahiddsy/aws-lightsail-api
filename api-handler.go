package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
)

type Location struct {
	AvailabilityZone string `json:"AvailabilityZone"`
	RegionName       string `json:"RegionName"`
}
type InstanceResponse struct {
	CreatedAt        string `json:"CreatedAt"`
	ErrorCode        string `json:"ErrorCode"`
	ErrorDetails     string `json:"ErrorDetails"`
	Id               string `json:"Id"`
	IsTerminal       bool   `json:"IsTerminal"`
	Location         Location
	OperationDetails string `json:"OperationDetails"`
	OperationType    string `json:"OperationType"`
	ResourceName     string `json:"ResourceName"`
	ResourceType     string `json:"ResourceType"`
	Status           string `json:"Status"`
	StatusChangedAt  string `json:"StatusChangedAt"`
}

var responses []InstanceResponse

var Regions = []string{
	"us-east-1", "us-east-2", "us-west-2",
	"eu-west-1", "eu-west-2", "eu-west-3",
	"eu-central-1", "ap-southeast-1",
	"ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
	"ap-south-1", "ca-central-1", "eu-north-1",
}

type Instance struct {
	Name     string `json:"Name"`
	State    string `json:"State"`
	PublicIP string `json:"PublicIP,omitempty"`
}

// statusInterceptor is a custom ResponseWriter that tracks the status code
type statusInterceptor struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader intercepts the status code before writing it to the response
func (i *statusInterceptor) WriteHeader(code int) {
	i.statusCode = code
	i.ResponseWriter.WriteHeader(code)
}

func logging(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Intercept the response writer to track the status code
		interceptor := &statusInterceptor{ResponseWriter: w}

		// Call the original handler with the intercepted writer
		f.ServeHTTP(interceptor, r)

		// Log the request details along with the status code
		log.Printf("- %s %s %s %s %s %s %d\n", r.RemoteAddr, r.Host, r.Method, r.URL, r.Proto, r.UserAgent(), interceptor.statusCode)
		//f.ServeHTTP(w, r)

	}
}

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

	instances := []Instance{}
	for _, instance := range resp.Instances {
		var inst Instance
		inst.Name = *instance.Name
		inst.State = *instance.State.Name

		if *instance.State.Name != "stopped" {
			inst.PublicIP = *instance.PublicIpAddress
		}

		instances = append(instances, inst)
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

	// Parse JSON into a slice of InstanceResponse
	err = json.Unmarshal([]byte(responseJSON), &responses)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
		return
	}

	if r.UserAgent() != "PostmanRuntime/7.35.0" {
		http.Redirect(w, r, "/api/status", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)

}

func listLightsailRegions(w http.ResponseWriter, r *http.Request) {

	// Convert Regions slice to JSON
	regionsJSON, err := json.Marshal(Regions)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response
	w.Write(regionsJSON)
}

func main() {
	http.HandleFunc("/api/instances", logging(listLightsailInstances))
	http.HandleFunc("/api/instance", logging(resetLightsailInstance))
	http.HandleFunc("/api/regions", logging(listLightsailRegions))

	http.HandleFunc("/api/status", logging(func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./web/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		//log.Printf("%s", responses[0])
		data := responses[0]

		// Execute the template and pass the response as data
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))

	fmt.Println("Server is running on :8080")
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
