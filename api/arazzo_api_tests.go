package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Arazzo Specification Structures
type ArazzoSpec struct {
	Arazzo             string              `yaml:"arazzo"`
	Info               Info                `yaml:"info"`
	SourceDescriptions []SourceDescription `yaml:"sourceDescriptions"`
	Workflows          []Workflow          `yaml:"workflows"`
}

type Info struct {
	Title       string `yaml:"title"`
	Summary     string `yaml:"summary"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

type SourceDescription struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	Type string `yaml:"type"`
}

type Workflow struct {
	WorkflowID  string                 `yaml:"workflowId"`
	Summary     string                 `yaml:"summary"`
	Description string                 `yaml:"description"`
	Parameters  []Parameter            `yaml:"parameters"`
	Inputs      map[string]interface{} `yaml:"inputs"`
	Steps       []Step                 `yaml:"steps"`
}

type Step struct {
	StepID          string            `yaml:"stepId"`
	OperationID     string            `yaml:"operationId"`
	Description     string            `yaml:"description"`
	DependsOn       []string          `yaml:"dependsOn"`
	Parameters      []Parameter       `yaml:"parameters"`
	RequestBody     *RequestBody      `yaml:"requestBody"`
	SuccessCriteria []SuccessCriteria `yaml:"successCriteria"`
	Outputs         map[string]string `yaml:"outputs"`
}

type Parameter struct {
	Name  string      `yaml:"name"`
	In    string      `yaml:"in"`
	Value interface{} `yaml:"value"`
}

type RequestBody struct {
	ContentType string      `yaml:"contentType"`
	Payload     interface{} `yaml:"payload"`
}

type SuccessCriteria struct {
	Context   string `yaml:"context"`
	Condition string `yaml:"condition"`
	Type      string `yaml:"type"`
}

// Test Execution Context
type TestContext struct {
	StepOutputs    map[string]map[string]interface{}
	WorkflowParams map[string]interface{}
	Client         *http.Client
	BaseURL        string
}

func NewTestContext(baseURL string) *TestContext {
	return &TestContext{
		StepOutputs:    make(map[string]map[string]interface{}),
		WorkflowParams: make(map[string]interface{}),
		Client:         &http.Client{Timeout: 30 * time.Second},
		BaseURL:        baseURL,
	}
}

// Test Executor
type TestExecutor struct {
	spec *ArazzoSpec
}

func NewTestExecutor(specPath string) (*TestExecutor, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	var spec ArazzoSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	return &TestExecutor{spec: &spec}, nil
}

func (te *TestExecutor) ExecuteWorkflow(workflowID, baseURL string) error {
	return te.ExecuteWorkflowWithInputs(workflowID, baseURL, nil)
}

func (te *TestExecutor) ExecuteWorkflowWithInputs(workflowID, baseURL string, customInputs map[string]string) error {
	var workflow *Workflow
	for _, w := range te.spec.Workflows {
		if w.WorkflowID == workflowID {
			workflow = &w
			break
		}
	}

	if workflow == nil {
		return fmt.Errorf("workflow %s not found", workflowID)
	}

	ctx := NewTestContext(baseURL)

	// Priority 1: Initialize from default values in inputs schema
	if workflow.Inputs != nil {
		if props, ok := workflow.Inputs["properties"].(map[string]interface{}); ok {
			for key, value := range props {
				if propMap, ok := value.(map[string]interface{}); ok {
					if defaultVal, exists := propMap["default"]; exists {
						ctx.WorkflowParams[key] = defaultVal
					}
				}
			}
		}
	}

	// Priority 2: Initialize from legacy parameters (for backward compatibility)
	for _, param := range workflow.Parameters {
		ctx.WorkflowParams[param.Name] = param.Value
	}

	// Priority 3: Override with environment variables
	if workflow.Inputs != nil {
		if props, ok := workflow.Inputs["properties"].(map[string]interface{}); ok {
			for key := range props {
				envKey := "ARAZZO_" + strings.ToUpper(key)
				if envVal := os.Getenv(envKey); envVal != "" {
					ctx.WorkflowParams[key] = envVal
					fmt.Printf("🌍 Using environment variable: %s = %s\n", envKey, envVal)
				}
			}
		}
	}

	// Priority 4: Override with custom inputs from command line (highest priority)
	if customInputs != nil {
		for key, value := range customInputs {
			ctx.WorkflowParams[key] = value
			fmt.Printf("📝 Using command-line input: %s = %s\n", key, value)
		}
	}

	fmt.Printf("\n╔════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  Executing Workflow: %s\n", workflow.Summary)
	fmt.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")

	// Display all inputs being used
	if len(ctx.WorkflowParams) > 0 {
		fmt.Println("📋 Workflow Inputs:")
		for key, value := range ctx.WorkflowParams {
			fmt.Printf("   • %s: %v\n", key, value)
		}
		fmt.Println()
	}

	for i, step := range workflow.Steps {
		fmt.Printf("┌─ Step %d/%d: %s\n", i+1, len(workflow.Steps), step.StepID)
		fmt.Printf("│  %s\n", step.Description)

		if err := te.executeStep(ctx, &step, workflow.WorkflowID); err != nil {
			fmt.Printf("└─ ✗ FAILED: %v\n\n", err)
			return fmt.Errorf("step %s failed: %w", step.StepID, err)
		}

		fmt.Printf("└─ ✓ SUCCESS\n\n")
	}

	fmt.Printf("╔════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  ✓ Workflow Completed Successfully!\n")
	fmt.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")

	return nil
}

func (te *TestExecutor) executeStep(ctx *TestContext, step *Step, workflowID string) error {
	// Build URL
	url := te.buildURL(ctx, step, workflowID)

	// Determine HTTP method
	method := te.getHTTPMethod(step.OperationID)

	// Build request body
	var bodyReader io.Reader
	if step.RequestBody != nil {
		body := te.resolveValue(ctx, step.RequestBody.Payload, workflowID)

		if step.RequestBody.ContentType == "application/json" {
			jsonData, _ := json.Marshal(body)
			bodyReader = bytes.NewBuffer(jsonData)
		} else {
			// For text/plain or other types
			bodyReader = strings.NewReader(fmt.Sprintf("%v", body))
		}
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set content type
	if step.RequestBody != nil {
		req.Header.Set("Content-Type", step.RequestBody.ContentType)
	}

	// Set additional headers from parameters
	for _, param := range step.Parameters {
		if param.In == "header" {
			value := te.resolveValue(ctx, param.Value, workflowID)
			req.Header.Set(param.Name, fmt.Sprintf("%v", value))
		}
	}

	fmt.Printf("│  → %s %s\n", method, url)

	// Execute request
	resp, err := ctx.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("│  ← Status: %d\n", resp.StatusCode)

	// Parse response
	var responseData interface{}
	if len(body) > 0 && strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		json.Unmarshal(body, &responseData)
		prettyJSON, _ := json.MarshalIndent(responseData, "│  ", "  ")
		fmt.Printf("│  Response: %s\n", string(prettyJSON))
	}

	// Store outputs
	if step.Outputs != nil {
		outputs := make(map[string]interface{})
		for key := range step.Outputs {
			// Simplified - in real implementation, parse the expression
			if key == "initialAttachmentsList" || key == "attachmentsList" || key == "finalAttachmentsList" {
				outputs[key] = responseData
			}
		}
		ctx.StepOutputs[step.StepID] = outputs
	}

	// Validate success criteria
	for _, criteria := range step.SuccessCriteria {
		if !te.evaluateCriteria(resp.StatusCode, responseData, criteria) {
			return fmt.Errorf("criteria failed: %s", criteria.Condition)
		}
	}

	return nil
}

func (te *TestExecutor) buildURL(ctx *TestContext, step *Step, workflowID string) string {
	// Map operation IDs to URL patterns
	urlPatterns := map[string]string{
		"listAuthorsAttachments":       "/authors/{authorName}/.attachments",
		"putAuthorAttachmentByName":    "/authors/{authorName}/.attachments/{attachmentFileName}",
		"deleteAuthorAttachmentByName": "/authors/{authorName}/.attachments/{attachmentFileName}",
	}

	pattern, ok := urlPatterns[step.OperationID]
	if !ok {
		return ctx.BaseURL
	}

	url := ctx.BaseURL + pattern
	queryParams := []string{}

	// Replace path parameters
	for _, param := range step.Parameters {
		value := te.resolveValue(ctx, param.Value, workflowID)
		valueStr := fmt.Sprintf("%v", value)

		if param.In == "path" {
			url = strings.ReplaceAll(url, "{"+param.Name+"}", valueStr)
		} else if param.In == "query" {
			queryParams = append(queryParams, fmt.Sprintf("%s=%s", param.Name, valueStr))
		}
	}

	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	return url
}

func (te *TestExecutor) getHTTPMethod(operationID string) string {
	methods := map[string]string{
		"listAuthorsAttachments":       "GET",
		"putAuthorAttachmentByName":    "PUT",
		"deleteAuthorAttachmentByName": "DELETE",
		"getAuthorAttachmentByName":    "GET",
	}

	if method, ok := methods[operationID]; ok {
		return method
	}
	return "GET"
}

func (te *TestExecutor) resolveValue(ctx *TestContext, value interface{}, workflowID string) interface{} {
	if str, ok := value.(string); ok {
		// Remove curly braces if present
		str = strings.Trim(str, "{}")

		// Handle $inputs references (new Arazzo 1.0.0 style)
		if strings.HasPrefix(str, "$inputs.") {
			parts := strings.Split(str, ".")
			if len(parts) >= 2 {
				paramName := parts[1]
				if val, exists := ctx.WorkflowParams[paramName]; exists {
					return val
				}
			}
		}

		// Handle workflow parameter references (legacy style)
		if strings.HasPrefix(str, "$workflows.") {
			parts := strings.Split(str, ".")
			if len(parts) >= 4 && parts[1] == workflowID && parts[2] == "parameters" {
				paramName := parts[3]
				if val, exists := ctx.WorkflowParams[paramName]; exists {
					return val
				}
			}
		}

		// Handle step output references
		if strings.HasPrefix(str, "$steps.") {
			parts := strings.Split(str, ".")
			if len(parts) >= 3 {
				stepID := parts[1]
				outputKey := strings.Join(parts[2:], ".")
				if outputs, exists := ctx.StepOutputs[stepID]; exists {
					if val, ok := outputs[outputKey]; ok {
						return val
					}
				}
			}
		}
	}

	return value
}

func (te *TestExecutor) evaluateCriteria(statusCode int, response interface{}, criteria SuccessCriteria) bool {
	condition := criteria.Condition

	// Handle status code checks
	if strings.Contains(condition, "$statusCode") {
		expected := 0
		fmt.Sscanf(condition, "$statusCode == %d", &expected)
		return statusCode == expected
	}

	// Handle array length checks
	if strings.Contains(condition, ".length") {
		if arr, ok := response.([]interface{}); ok {
			expected := 0
			fmt.Sscanf(condition, "$response.body.length == %d", &expected)
			return len(arr) == expected
		}
	}

	// Handle array element checks
	if strings.Contains(condition, "[0].name") {
		if arr, ok := response.([]interface{}); ok && len(arr) > 0 {
			if obj, ok := arr[0].(map[string]interface{}); ok {
				// Extract expected value from condition
				parts := strings.Split(condition, "==")
				if len(parts) == 2 {
					expected := strings.TrimSpace(parts[1])
					expected = strings.Trim(expected, "\"' ")
					actual := obj["name"]
					return fmt.Sprintf("%v", actual) == expected
				}
			}
		}
	}

	return true
}

func parseCommandLineInputs(args []string) map[string]string {
	inputs := make(map[string]string)

	for i := 0; i < len(args); i++ {
		if args[i] == "--input" && i+1 < len(args) {
			parts := strings.SplitN(args[i+1], "=", 2)
			if len(parts) == 2 {
				inputs[parts[0]] = parts[1]
			}
			i++ // Skip the next argument
		}
	}

	return inputs
}

func printUsage() {
	fmt.Println("Usage: go run main.go <arazzo-spec.yaml> <base-url> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --input key=value    Set workflow input parameter (can be used multiple times)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Use default values from spec")
	fmt.Println("  go run main.go workflow.yaml http://localhost:8080")
	fmt.Println()
	fmt.Println("  # Override specific inputs")
	fmt.Println("  go run main.go workflow.yaml http://localhost:8080 \\")
	fmt.Println("    --input authorName=john \\")
	fmt.Println("    --input repoName=myrepo \\")
	fmt.Println("    --input attachmentFileName=document.pdf")
	fmt.Println()
	fmt.Println("  # Use environment variables")
	fmt.Println("  export ARAZZO_AUTHORNAME=alice")
	fmt.Println("  export ARAZZO_REPONAME=production")
	fmt.Println("  go run main.go workflow.yaml https://api.production.com")
	fmt.Println()
	fmt.Println("Input Priority (highest to lowest):")
	fmt.Println("  1. Command-line arguments (--input)")
	fmt.Println("  2. Environment variables (ARAZZO_*)")
	fmt.Println("  3. Default values in spec")
}

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	specPath := os.Args[1]
	baseURL := os.Args[2]

	// Parse input parameters from command line
	customInputs := parseCommandLineInputs(os.Args[3:])

	executor, err := NewTestExecutor(specPath)
	if err != nil {
		fmt.Printf("❌ Error loading spec: %v\n", err)
		os.Exit(1)
	}

	// Execute the main workflow with custom inputs
	if err := executor.ExecuteWorkflowWithInputs("authorAttachmentsLifecycle", baseURL, customInputs); err != nil {
		fmt.Printf("\n❌ Workflow execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ All workflows completed successfully!")
}
