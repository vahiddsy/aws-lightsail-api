package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
	"github.com/cloudflare/cloudflare-go"
)

type CFconfig struct {
	ApiKey   string `json:"apikey"`
	ApiEmail string `json:"apiemail"`
	Domain   string `json:"domain"`
}

var cloud_config []CFconfig

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

func changeIpInstance(region, profile, instanceName string) (*lightsail.GetStaticIpOutput, error) {
	client, err := credentialing(region, profile)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	getStaticIPsInput := &lightsail.GetStaticIpsInput{
		//IncludeStaticIpMetadata: aws.Bool(true), // IncludeStaticIpMetadata should be set to true
	}

	getStaticIPsOutput, err := client.GetStaticIps(context.TODO(), getStaticIPsInput) // Method name is GetStaticIps
	if err != nil {
		return nil, fmt.Errorf("Error Get Static IPs: %v", err)
	}

	// Release any existing static IP associated with the instance
	var oldIpInstance string
	//var newIpInstance string
	for _, ip := range getStaticIPsOutput.StaticIps {
		if ip.AttachedTo != nil && *ip.AttachedTo == instanceName {
			releaseStaticIPInput := &lightsail.ReleaseStaticIpInput{
				StaticIpName: ip.Name,
			}
			//inja mitoni ip ghabli ro ghable release to value bezari
			oldIpInstance = *ip.IpAddress
			_, err = client.ReleaseStaticIp(context.TODO(), releaseStaticIPInput)
			if err != nil {
				return nil, fmt.Errorf("Error Release Static IP: %v", err)
			}
		}
	}

	createStaticIPInput := &lightsail.AllocateStaticIpInput{
		StaticIpName: aws.String(fmt.Sprintf("IP-%s", instanceName)), // Corrected
	}

	createStaticIPOutput, err := client.AllocateStaticIp(context.TODO(), createStaticIPInput)
	if err != nil {
		return nil, fmt.Errorf("Error Create Static IP : %v", err)
	}

	// Attach the static IP to the instance
	attachStaticIPInput := &lightsail.AttachStaticIpInput{
		//StaticIpName: aws.String(fmt.Sprintf("IP-%s", instanceName)),
		StaticIpName: createStaticIPOutput.Operations[0].ResourceName,
		InstanceName: aws.String(instanceName), // Corrected
	}
	_, err = client.AttachStaticIp(context.TODO(), attachStaticIPInput)
	if err != nil {
		return nil, fmt.Errorf("Error Attach Static IP instance: %v", err)
	}

	getStaticIPInput := &lightsail.GetStaticIpInput{
		StaticIpName: createStaticIPOutput.Operations[0].ResourceName,
	}

	getStaticIPOutput, err := client.GetStaticIp(context.TODO(), getStaticIPInput) // Method name is GetStaticIps
	if err != nil {
		return nil, fmt.Errorf("Error Get Static IP: %v", err)
	}

	//Update DNS Record
	//newIp := getStaticIPOutput.StaticIp.IpAddress
	newIpInstance := *getStaticIPOutput.StaticIp.IpAddress
	if profile == "vahid" {
		// apiKey := os.Getenv("CF_API_KEY")
		// apiEmail := os.Getenv("CF_API_EMAIL")
		apiKey := cloud_config[0].ApiKey
		apiEmail := cloud_config[0].ApiEmail
		//zoneID := os.Getenv("YOUR_ZONE_ID") // Replace with your Cloudflare zone ID
		// dnsRecordID := "YOUR_DNS_RECORD_ID" // Replace with the ID of the DNS record you want to update
		// newIP := "NEW_IP_ADDRESS"           // Replace with the new IP address
		// Create a new Cloudflare API client
		api, err := cloudflare.New(apiKey, apiEmail)
		if err != nil {
			fmt.Println("Error creating Cloudflare API client:", err)
		}

		zoneID, err := api.ZoneIDByName(cloud_config[0].Domain)
		if err != nil {
			log.Printf("Error in ZoneID  : %v\n", err)
		}
		recs, _, err := api.ListDNSRecords(context.Background(), cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{Type: "A", Content: oldIpInstance})
		if err != nil {
			log.Printf("Error in List DNS Record :  %v\n", err)
		}

		for _, r := range recs {
			log.Printf("Record is  : %s: %s , %s , %s\n", r.Name, r.Content, r.ID, r.Comment)
			params := cloudflare.UpdateDNSRecordParams{ID: r.ID, Type: "A", Name: r.Name, Content: newIpInstance, TTL: 1}
			response, err := api.UpdateDNSRecord(context.Background(), cloudflare.ZoneIdentifier(zoneID), params)
			if err != nil {
				log.Printf("Error Get DNS Record : %v\n", err)
				//log.Fatal(err)
			}
			log.Printf("Response Success GET DNS Record is %v\n", response)

		}
	}

	return getStaticIPOutput, nil
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

func actionLightsailInstance(w http.ResponseWriter, r *http.Request) {
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
	if now-timestamp > 3600 {
		http.Error(w, "Invalid request: Timestamp is more than 1 hours difference from current time", http.StatusBadRequest)
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

	} else if action == "changeip" {
		changeResp, err := changeIpInstance(region, profile, instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error change IP instance status: %v", err), http.StatusInternalServerError)
			return
		}
		resp = changeResp

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

	configFile := flag.String("config", "./config.json", "string")
	flag.Parse()

	// Read JSON file
	file_config, err := os.Open(*configFile)
	if err != nil {
		log.Println("Error opening config file:", err)
		return
	}
	defer file_config.Close()

	// Decode JSON data into slice of Config structs
	//var config []Config
	decoder_config := json.NewDecoder(file_config)
	if err := decoder_config.Decode(&cloud_config); err != nil {
		log.Println("Error decoding config JSON:", err)
		return
	}
	fmt.Printf("Load Cloudflare Config...\n%v\n", cloud_config)

	http.HandleFunc("/api/instances", logging(listLightsailInstances))
	http.HandleFunc("/api/instance", logging(actionLightsailInstance))
	http.HandleFunc("/api/regions", logging(listLightsailRegions))

	// http.HandleFunc("/api/status", logging(func(w http.ResponseWriter, r *http.Request) {
	// 	tmpl, err := template.ParseFiles("./web/index.html")
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}
	// 	//log.Printf("%s", responses[0])
	// 	data := responses[0]

	// 	// Execute the template and pass the response as data
	// 	err = tmpl.Execute(w, data)
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}
	// }))

	fmt.Println("Version : v1.6\nServer is running on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
