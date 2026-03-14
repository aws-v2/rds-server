package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type CreateInstanceInput struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type InstanceOutput struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Engine    string    `json:"engine"`
	Port      int       `json:"port"`
	User      string    `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
}

type ListInstancesOutput struct {
	Instances []InstanceOutput `json:"instances"`
}

func getCredentials() (string, string, error) {
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if accessKeyID == "" || secretAccessKey == "" {
		return "", "", fmt.Errorf("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables must be set")
	}

	return accessKeyID, secretAccessKey, nil
}

func getBaseURL() string {
	baseURL := os.Getenv("RDS_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8087"
	}
	return baseURL
}

func makeRequest(method, url string, body io.Reader, accessKeyID, secretAccessKey string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", fmt.Sprintf("%s:%s", accessKeyID, secretAccessKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

func createDB() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: miniaws rds create-db <name> <user> <password>")
		return
	}

	name := os.Args[2]
	user := os.Args[3]
	password := os.Args[4]

	accessKeyID, secretAccessKey, err := getCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	input := CreateInstanceInput{
		Name:     name,
		User:     user,
		Password: password,
	}

	jsonData, err := json.Marshal(input)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		return
	}

	baseURL := getBaseURL()
	resp, err := makeRequest("POST", baseURL+"/api/v1/rds/instances", bytes.NewBuffer(jsonData), accessKeyID, secretAccessKey)
	if err != nil {
		fmt.Printf("Error creating DB: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var instance InstanceOutput
		if err := json.Unmarshal(body, &instance); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			return
		}
		fmt.Printf("Database instance created successfully!\n")
		fmt.Printf("ID: %s\n", instance.ID)
		fmt.Printf("Name: %s\n", instance.Name)
		fmt.Printf("Engine: %s\n", instance.Engine)
		fmt.Printf("Port: %d\n", instance.Port)
		fmt.Printf("User: %s\n", instance.User)
		fmt.Printf("CreatedAt: %s\n", instance.CreatedAt.Format(time.RFC3339))
	} else {
		fmt.Printf("Failed to create DB instance. Status: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func listDB() {
	accessKeyID, secretAccessKey, err := getCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	baseURL := getBaseURL()
	resp, err := makeRequest("GET", baseURL+"/api/v1/rds/instances", nil, accessKeyID, secretAccessKey)
	if err != nil {
		fmt.Printf("Error listing DBs: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var output ListInstancesOutput
		if err := json.Unmarshal(body, &output); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			return
		}

		if len(output.Instances) == 0 {
			fmt.Println("No database instances found.")
			return
		}

		fmt.Printf("Found %d database instance(s):\n", len(output.Instances))
		for _, inst := range output.Instances {
			fmt.Printf("\nID: %s\n", inst.ID)
			fmt.Printf("Name: %s\n", inst.Name)
			fmt.Printf("Engine: %s\n", inst.Engine)
			fmt.Printf("Port: %d\n", inst.Port)
			fmt.Printf("User: %s\n", inst.User)
			fmt.Printf("CreatedAt: %s\n", inst.CreatedAt.Format(time.RFC3339))
		}
	} else {
		fmt.Printf("Failed to list DB instances. Status: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func describeDB() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: miniaws rds describe-db <instance-id>")
		return
	}

	instanceID := os.Args[2]

	accessKeyID, secretAccessKey, err := getCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	baseURL := getBaseURL()
	resp, err := makeRequest("GET", baseURL+"/api/v1/rds/instances/"+instanceID, nil, accessKeyID, secretAccessKey)
	if err != nil {
		fmt.Printf("Error describing DB: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var instance InstanceOutput
		if err := json.Unmarshal(body, &instance); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			return
		}
		fmt.Printf("Database Instance Details:\n")
		fmt.Printf("ID: %s\n", instance.ID)
		fmt.Printf("Name: %s\n", instance.Name)
		fmt.Printf("Engine: %s\n", instance.Engine)
		fmt.Printf("Port: %d\n", instance.Port)
		fmt.Printf("User: %s\n", instance.User)
		fmt.Printf("CreatedAt: %s\n", instance.CreatedAt.Format(time.RFC3339))
	} else {
		fmt.Printf("Failed to describe DB instance. Status: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func deleteDB() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: miniaws rds delete-db <instance-id>")
		return
	}

	instanceID := os.Args[2]

	accessKeyID, secretAccessKey, err := getCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	baseURL := getBaseURL()
	resp, err := makeRequest("DELETE", baseURL+"/api/v1/rds/instances/"+instanceID, nil, accessKeyID, secretAccessKey)
	if err != nil {
		fmt.Printf("Error deleting DB: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Database instance %s deleted successfully!\n", instanceID)
	} else {
		fmt.Printf("Failed to delete DB instance. Status: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func startDB() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: miniaws rds start-db <instance-id>")
		return
	}

	instanceID := os.Args[2]

	accessKeyID, secretAccessKey, err := getCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	baseURL := getBaseURL()
	resp, err := makeRequest("POST", baseURL+"/api/v1/rds/instances/"+instanceID+"/start", nil, accessKeyID, secretAccessKey)
	if err != nil {
		fmt.Printf("Error starting DB: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Database instance %s started successfully!\n", instanceID)
	} else {
		fmt.Printf("Failed to start DB instance. Status: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func stopDB() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: miniaws rds stop-db <instance-id>")
		return
	}

	instanceID := os.Args[2]

	accessKeyID, secretAccessKey, err := getCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	baseURL := getBaseURL()
	resp, err := makeRequest("POST", baseURL+"/api/v1/rds/instances/"+instanceID+"/stop", nil, accessKeyID, secretAccessKey)
	if err != nil {
		fmt.Printf("Error stopping DB: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Database instance %s stopped successfully!\n", instanceID)
	} else {
		fmt.Printf("Failed to stop DB instance. Status: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func viewLogs() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: miniaws rds logs <instance-id>")
		return
	}

	instanceID := os.Args[2]

	accessKeyID, secretAccessKey, err := getCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	baseURL := getBaseURL()
	resp, err := makeRequest("GET", baseURL+"/api/v1/rds/instances/"+instanceID+"/logs", nil, accessKeyID, secretAccessKey)
	if err != nil {
		fmt.Printf("Error viewing logs: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Audit logs for instance %s:\n", instanceID)
		fmt.Println(string(body))
	} else {
		fmt.Printf("Failed to view logs. Status: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: miniaws rds <command>")
		fmt.Println("Commands:")
		fmt.Println("  create-db <name> <user> <password> - Create a new database instance")
		fmt.Println("  list-db                             - List all database instances")
		fmt.Println("  describe-db <instance-id>           - Describe a specific database instance")
		fmt.Println("  delete-db <instance-id>             - Delete a database instance")
		fmt.Println("  start-db <instance-id>              - Start a stopped database instance")
		fmt.Println("  stop-db <instance-id>               - Stop a running database instance")
		fmt.Println("  logs <instance-id>                  - View audit logs for an instance")
		return
	}

	command := os.Args[1]
	switch command {
	case "create-db":
		createDB()
	case "list-db":
		listDB()
	case "describe-db":
		describeDB()
	case "delete-db":
		deleteDB()
	case "start-db":
		startDB()
	case "stop-db":
		stopDB()
	case "logs":
		viewLogs()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: create-db, list-db, describe-db, delete-db, start-db, stop-db, logs")
	}
}
