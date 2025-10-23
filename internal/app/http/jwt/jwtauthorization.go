package jwt

import (
	"context"
	"fmt"
	"log"
	"os"

	"encoding/json"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
)

type UserGuard map[string]string

type GuardDefinition struct {
	Guards     UserGuard `json:"guards"`
	Inventory  bool      `json:"inventory"`
	Namespaces []string  `json:"namespaces"`
	Operations []string  `json:"operations"`
}

type GuardConfig []GuardDefinition

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

func init() {
	if err := godotenv.Load("auth.env"); err != nil {
		log.Printf("Warning: auth.env file not found")
	}
}

func ValidateJWT(token string, ac *AccessControl) (bool, map[string]string) {
	matchedGuards := make(map[string]string)
	claims, err := parseToken(token)
	if err != nil {
		log.Printf("Failed to parse token: %v", err)
		return false, nil
	}

	fmt.Println("--- Decoded JWT Claims (fmt.Println) ---")
	fmt.Println(claims)
	fmt.Println("---------------------------------------")

	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < time.Now().Unix() {
			log.Printf("Token has expired.")
			return false, nil
		}
	} else {
		log.Printf("Warning: 'exp' claim not found or not a valid number in token. Proceeding without expiration check.")
	}
	whitelistFile, err := os.ReadFile(ac.filePath)
	if err != nil {
		log.Printf("Error reading whitelist config")
		return false, nil
	}
	requiredGuards, err := loadGuardsFromJSON(whitelistFile)
	if err != nil {
		log.Println("failed to unmarshal guards JSON: %w", err)
		return false, nil
	}
	if len(requiredGuards) == 0 {
		log.Printf("Token validated successfully for user (no guards required).")
		return true, matchedGuards
	}

	guardFound := false
	for i, def := range requiredGuards {
		log.Printf("Checking Guard Definition #%d...", i+1)

		if len(def.Guards) == 0 {
			log.Printf("Guard Definition #%d has no specific 'guards' fields to check. Skipping this definition.", i+1)
			continue
		}

		currentDefinitionGuardMatch := false
		for guardKey, expectedGuardValue := range def.Guards {
			if claimValue, ok := claims[guardKey]; ok {
				if strClaimValue, isString := claimValue.(string); isString {
					if strClaimValue == expectedGuardValue {
						log.Printf("Guard Definition #%d: Claim '%s' in token ('%s') matched expected value ('%s'). This definition is satisfied!", i+1, guardKey, strClaimValue, expectedGuardValue)
						matchedGuards[guardKey] = strClaimValue
						currentDefinitionGuardMatch = true
						break
					}
				} else {
					log.Printf("Warning: Guard Definition #%d: Claim '%s' in token is not a string (it's %T). Skipping comparison for this guard.", i+1, guardKey, claimValue)
				}
			} else {
				log.Printf("Guard Definition #%d: Guard key '%s' not found in token claims. Skipping this guard.", i+1, guardKey)
			}
		}

		if currentDefinitionGuardMatch {
			guardFound = true
			log.Printf("Found a matching Guard Definition (#%d). Token is authorized!", i+1)
			break
		}
	}

	if !guardFound {
		log.Printf("Token failed guard validation: No matching guard definition found among the provided configurations.")
		return false, nil
	}

	log.Printf("Token validated successfully for user (including guard checks).")
	return true, matchedGuards
}

func parseToken(tokenString string) (jwt.MapClaims, error) {
	parser := new(jwt.Parser)
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}
	return claims, nil
}

func loadGuardsFromJSON(jsonString []byte) (GuardConfig, error) {
	var guards GuardConfig
	err := json.Unmarshal(jsonString, &guards)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal guards JSON: %w", err)
	}
	return guards, nil
}

func NewAccessControl(whitelistPath string) (*AccessControl, error) {
	ac := &AccessControl{
		filePath: whitelistPath,
	}

	if err := ac.reload(); err != nil {
		return nil, err
	}

	if err := ac.ValidateRules(); err != nil {
		return nil, fmt.Errorf("invalid rules: %w", err)
	}

	return ac, nil
}

func (ac *AccessControl) reload() error {
	content, err := os.ReadFile(ac.filePath)
	if err != nil {
		return fmt.Errorf("failed to read whitelist file: %w", err)
	}

	var rules []AccessRule
	if err := json.Unmarshal(content, &rules); err != nil {
		return fmt.Errorf("failed to parse whitelist JSON: %w", err)
	}

	ac.Rules = rules
	ac.lastRead = time.Now()
	return nil
}

func (ac *AccessControl) HasAccess(userClaims map[string]string, namespace string, r *http.Request) bool {
	var operation Operation
	switch r.Method {
	case http.MethodGet:
		operation = OpGet
	case http.MethodPost:
		operation = OpPost
	case http.MethodPut:
		operation = OpPut
	case http.MethodDelete:
		operation = OpDelete
	default:
		fmt.Printf("This is a %s request\n It's handling in authorization logic hasn't been implemented yet", r.Method)
		return false
	}
	if ac.shouldReload() {
		if err := ac.reload(); err != nil {
			log.Printf("Failed to reload whitelist: %v", err)
		}
	}

	for _, rule := range ac.Rules {
		guardMatched := false
		if len(rule.Guards) == 0 {
			log.Printf("Rule has no guards defined, applies to all users. Rule: %+v", rule)
			guardMatched = true
		} else {
			for guardKey, guardValue := range rule.Guards {

				if userClaimValue, ok := userClaims[guardKey]; ok && userClaimValue == guardValue {
					guardMatched = true
					break
				}
			}
		}

		if !guardMatched {
			continue
		}

		//Chcek if iternary access is allowed
		if rule.Inventory && namespace == "inventory" {
			return true
		}

		// Check if namespace matches any of the allowed namespaces
		namespaceAllowed := false
		for _, allowedNs := range rule.Namespaces {
			if allowedNs == "*" || allowedNs == namespace {
				namespaceAllowed = true
				break
			}
		}
		if !namespaceAllowed {
			continue
		}

		// Check if operation is allowed
		for _, allowedOp := range rule.Operations {
			if allowedOp == operation || allowedOp == "*" {
				return true
			}
		}
	}

	return false
}

func (ac *AccessControl) ValidateRules() error {
	for i, rule := range ac.Rules {
		// Validate guards
		if len(rule.Guards) == 0 {
			return fmt.Errorf("rule %d: 'guards' field is missing or empty. At least one guard key-value pair is required.", i)
		}
		for key, value := range rule.Guards {
			if key == "" {
				return fmt.Errorf("rule %d: empty guard key found", i)
			}
			if value == "" {
				return fmt.Errorf("rule %d: empty guard value for key '%s'", i, key)
			}
		}

		// Validate namespaces
		if len(rule.Namespaces) == 0 {
			return fmt.Errorf("rule %d: no namespaces specified", i)
		}
		for j, ns := range rule.Namespaces {
			if ns == "" {
				return fmt.Errorf("rule %d: empty namespace at index %d", i, j)
			}
		}

		// Validate operations
		if len(rule.Operations) == 0 {
			return fmt.Errorf("rule %d: no operations specified", i)
		}
		for j, op := range rule.Operations {
			switch op {
			case OpGet, OpPost, OpPut, OpDelete, "*":
				// valid operation
			default:
				return fmt.Errorf("rule %d: invalid operation at index %d: %s", i, j, op)
			}
		}
	}
	return nil
}

func (ac *AccessControl) shouldReload() bool {
	info, err := os.Stat(ac.filePath)
	if err != nil {
		return false
	}
	return info.ModTime().After(ac.lastRead)
}

func (ac *AccessControl) ListUserAccess(userClaims map[string]string) []AccessRule {
	var rules []AccessRule
	for _, rule := range ac.Rules {
		guardMatched := false
		if len(rule.Guards) == 0 {
			guardMatched = true // Applies to all if no specific guards
		} else {
			for guardKey, guardValue := range rule.Guards {
				if userClaimValue, ok := userClaims[guardKey]; ok && userClaimValue == guardValue {
					guardMatched = true
					break
				}
			}
		}

		if guardMatched {
			rules = append(rules, rule)
		}
	}
	return rules
}

func (ac *AccessControl) GetUserNamespaces(userClaims map[string]string) []string {
	namespaces := make(map[string]bool)
	for _, rule := range ac.Rules {
		guardMatched := false
		if len(rule.Guards) == 0 {
			guardMatched = true
		} else {
			for guardKey, guardValue := range rule.Guards {
				if userClaimValue, ok := userClaims[guardKey]; ok && userClaimValue == guardValue {
					guardMatched = true
					break
				}
			}
		}

		if guardMatched {
			for _, ns := range rule.Namespaces {
				namespaces[ns] = true
			}
		}
	}

	result := make([]string, 0, len(namespaces))
	for ns := range namespaces {
		result = append(result, ns)
	}
	return result
}

func (ac *AccessControl) GetUserOperations(userClaims map[string]string, namespace string) []Operation {
	operations := make(map[Operation]bool)
	for _, rule := range ac.Rules {
		guardMatched := false
		if len(rule.Guards) == 0 {
			guardMatched = true
		} else {
			for guardKey, guardValue := range rule.Guards {
				if userClaimValue, ok := userClaims[guardKey]; ok && userClaimValue == guardValue {
					guardMatched = true
					break
				}
			}
		}

		if guardMatched {
			for _, allowedNs := range rule.Namespaces {
				if allowedNs == namespace || allowedNs == "*" { // Check if namespace matches or is wildcard
					for _, op := range rule.Operations {
						operations[op] = true
					}
				}
			}
		}
	}

	result := make([]Operation, 0, len(operations))
	for op := range operations {
		result = append(result, op)
	}
	return result
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
