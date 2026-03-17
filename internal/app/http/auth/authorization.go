package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type AccessScopes struct {
	AllowedScopes []string `json:"scopes"`
}

type AccessControl struct {
	Rules    []AccessScopes
	filePath string
	lastRead time.Time
}

var GlobalAccessControl *AccessControl

func InitializeAccessControl(filePath string) error {
	if GlobalAccessControl == nil {
		GlobalAccessControl = &AccessControl{}
	}
	GlobalAccessControl.filePath = filePath
	return GlobalAccessControl.reload()
}

func LoadScopesFromJSON(jsonString []byte) (AccessScopes, error) {
	var scopes AccessScopes
	err := json.Unmarshal(jsonString, &scopes)
	if err != nil {
		return scopes, fmt.Errorf("failed to unmarshal scopes JSON: %w", err)
	}
	return scopes, nil
}

func (ac *AccessControl) reload() error {
	content, err := os.ReadFile(GlobalAccessControl.filePath)
	if err != nil {

		return fmt.Errorf("failed to read whitelist file %s: %w", GlobalAccessControl.filePath, err)
	}

	var rules []AccessScopes
	if err := json.Unmarshal(content, &rules); err != nil {
		return fmt.Errorf("failed to parse whitelist JSON: %w", err)
	}

	ac.Rules = rules
	ac.lastRead = time.Now()
	return nil
}

func (ac *AccessControl) shouldReload() bool {
	info, err := os.Stat(ac.filePath)
	if err != nil {
		return false
	}
	return info.ModTime().After(ac.lastRead)
}
