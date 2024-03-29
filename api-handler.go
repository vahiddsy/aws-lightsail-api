package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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
		log.Printf("- %s %s - %d %s %s - %s '%s'\n", r.RemoteAddr, r.Host, interceptor.statusCode, r.Method, r.URL, r.Proto, r.UserAgent())
		//f.ServeHTTP(w, r)

	}
}

func credentialing(region, profile string) (*lightsail.Client, error) {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithSharedConfigFiles([]string{"aws/config"}),
		config.WithSharedCredentialsFiles([]string{"aws/credentials"}),
		config.WithSharedConfigProfile(profile),
		config.WithRegion(region),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	client := lightsail.NewFromConfig(cfg)
	return client, nil
}

func resetInstance(region, profile, instanceName string) (*lightsail.RebootInstanceOutput, error) {

	client, err := credentialing(region, profile)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Use the "name" query parameter to identify the instance
	params := &lightsail.RebootInstanceInput{
		InstanceName: aws.String(instanceName),
	}
	resp, err := client.RebootInstance(context.TODO(), params)
	if err != nil {
		return nil, fmt.Errorf("Error getting instance: %v", err)
	}

	return resp, nil
}

func statusInstance(region, profile, instanceName string) (*lightsail.GetInstanceOutput, error) {
	client, err := credentialing(region, profile)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	params := &lightsail.GetInstanceInput{
		InstanceName: aws.String(instanceName),
	}

	resp, err := client.GetInstance(context.TODO(), params)
	if err != nil {
		return nil, fmt.Errorf("Error getting instance status: %v", err)
	}

	return resp, nil
}

func powerOffInstance(region, profile, instanceName string) (*lightsail.StopInstanceOutput, error) {
	client, err := credentialing(region, profile)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	params := &lightsail.StopInstanceInput{
		InstanceName: aws.String(instanceName),
	}

	resp, err := client.StopInstance(context.TODO(), params)
	if err != nil {
		return nil, fmt.Errorf("Error stopping instance: %v", err)
	}

	return resp, nil
}

func powerOnInstance(region, profile, instanceName string) (*lightsail.StartInstanceOutput, error) {
	client, err := credentialing(region, profile)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	params := &lightsail.StartInstanceInput{
		InstanceName: aws.String(instanceName),
	}

	resp, err := client.StartInstance(context.TODO(), params)
	if err != nil {
		return nil, fmt.Errorf("Error starting instance: %v", err)
	}

	return resp, nil
}

func listLightsailInstances(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	profile := r.URL.Query().Get("profile")
	if region == "" || profile == "" {
		http.Error(w, "Please provide a 'region' or 'profile' query parameter", http.StatusBadRequest)
		return
	}

	client, err := credentialing(region, profile)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading AWS configuration: %v", err), http.StatusInternalServerError)
		return
	}

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
	// Parse Profile query parameter
	profile := r.URL.Query().Get("profile")
	// Parse Action query parameter
	action := r.URL.Query().Get("action")

	if region == "" || instanceName == "" || timestampParam == "" || profile == "" || action == "" {
		http.Error(w, "Please provide 'region' , 'name' , 'secret' , 'profile' and 'action' query parameters", http.StatusBadRequest)
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

	var resp interface{}

	if action == "reset" {
		resetResp, err := resetInstance(region, profile, instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reseting instance: %v", err), http.StatusInternalServerError)
			return
		}
		resp = resetResp
	} else if action == "poweroff" {
		powerOffResp, err := powerOffInstance(region, profile, instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error powering off instance: %v", err), http.StatusInternalServerError)
			return
		}
		resp = powerOffResp
	} else if action == "poweron" {
		powerOnResp, err := powerOnInstance(region, profile, instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error powering on instance: %v", err), http.StatusInternalServerError)
			return
		}
		resp = powerOnResp

	} else if action == "status" {
		statusResp, err := statusInstance(region, profile, instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting instance status: %v", err), http.StatusInternalServerError)
			return
		}
		resp = statusResp
	} else {
		http.Error(w, "Invalid action specified", http.StatusBadRequest)
		return
	}

	// Add the reset logic here
	// You can use the "name" query parameter to reset the specific instance

	responseJSON, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse JSON into a slice of InstanceResponse
	// err = json.Unmarshal([]byte(responseJSON), &responses)
	// if err != nil {
	// 	http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
	// 	return
	// }

	// if r.UserAgent() == "Antinone" {
	// 	http.Redirect(w, r, "/api/status", http.StatusSeeOther)
	// 	return
	// }
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
	w.WriteHeader(http.StatusOK)
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

	fmt.Println("Version : v1.4\nServer is running on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
