package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type GuardConfig []AccessRule

type Operation string

const (
	OpGet    Operation = "GET"
	OpPost   Operation = "POST"
	OpPut    Operation = "PUT"
	OpDelete Operation = "DELETE"
)

type AccessRule struct {
	Guards     map[string]string `json:"guards"`
	Namespaces []string          `json:"namespaces"`
	Operations []Operation       `json:"operations"`
	Inventory  bool              `json:"inventory"`
}

type AccessControl struct {
	Rules    []AccessRule
	filePath string
	lastRead time.Time
}

var GlobalAccessControl *AccessControl

func InitializeAccessControl() error {
	if GlobalAccessControl == nil {
		GlobalAccessControl = &AccessControl{}
	}
	return GlobalAccessControl.reload()
}

func LoadGuardsFromJSON(jsonString []byte) (GuardConfig, error) {
	var guards GuardConfig
	err := json.Unmarshal(jsonString, &guards)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal guards JSON: %w", err)
	}
	return guards, nil
}

func (ac *AccessControl) reload() error {
	content, err := os.ReadFile(GlobalAccessControl.filePath)
	if err != nil {

		return fmt.Errorf("failed to read whitelist file %s: %w", GlobalAccessControl.filePath, err)
	}

	var rules []AccessRule
	if err := json.Unmarshal(content, &rules); err != nil {
		return fmt.Errorf("failed to parse whitelist JSON: %w", err)
	}

	ac.Rules = rules
	ac.lastRead = time.Now()
	return nil
}

func GetCLIToken() string {
	credential, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		log.Fatalf("Failed to create credential: %v", err)
	}

	token, err := credential.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	})
	if err != nil {
		log.Printf("Not logged in. Please run 'az login' first: %v", err)
		return ""
	}

	fmt.Println("Successfully authenticated!")
	fmt.Printf("Token: %s\n", token.Token)

	return token.Token
}
