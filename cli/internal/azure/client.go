package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

const (
	apiVersion = "2022-08-01"
	baseURL    = "https://management.azure.com"
)

type Client struct {
	subscriptionID string
	resourceGroup  string
	apimName       string
	accessToken    string
	httpClient     *http.Client
	verbose        bool
}

func NewClient(subscriptionID, resourceGroup, apimName string, verbose bool) (*Client, error) {
	token, err := getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure access token: %w\nRun 'az login' first", err)
	}

	if subscriptionID == "" {
		subscriptionID, err = getSubscriptionID()
		if err != nil {
			return nil, fmt.Errorf("failed to get subscription ID: %w", err)
		}
	}

	return &Client{
		subscriptionID: subscriptionID,
		resourceGroup:  resourceGroup,
		apimName:       apimName,
		accessToken:    token,
		verbose:        verbose,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func getAccessToken() (string, error) {
	cmd := exec.Command("az", "account", "get-access-token", "--query", "accessToken", "-o", "tsv")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getSubscriptionID() (string, error) {
	cmd := exec.Command("az", "account", "show", "--query", "id", "-o", "tsv")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *Client) request(method, path string, body any) ([]byte, int, error) {
	respBody, status, _, err := c.requestWithHeaders(method, path, body)
	return respBody, status, err
}

func (c *Client) requestWithHeaders(method, path string, body any) ([]byte, int, http.Header, error) {
	return c.requestWithHeadersAndQuery(method, path, nil, body)
}

func (c *Client) requestWithHeadersAndQuery(method, path string, query url.Values, body any) ([]byte, int, http.Header, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("api-version", apiVersion)
	requestURL := fmt.Sprintf("%s%s?%s", baseURL, path, query.Encode())

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, requestURL, bodyReader)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if c.verbose {
		c.logResponse(method, requestURL, resp.StatusCode, resp.Header, respBody)
	}

	return respBody, resp.StatusCode, resp.Header, nil
}

func (c *Client) logResponse(method, url string, status int, headers http.Header, body []byte) {
	fmt.Printf("\n[azure] %s %s\n", method, url)
	fmt.Printf("[azure] status: %d\n", status)
	if len(headers) > 0 {
		fmt.Printf("[azure] headers:\n")
		for key, values := range headers {
			fmt.Printf("[azure]   %s: %s\n", key, strings.Join(values, ", "))
		}
	}
	fmt.Printf("[azure] response: %s\n", string(body))
}

func (c *Client) apimPath(resource string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ApiManagement/service/%s%s",
		c.subscriptionID, c.resourceGroup, c.apimName, resource)
}

type API struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	DisplayName          string `json:"displayName"`
	Path                 string `json:"path"`
	ServiceURL           string `json:"serviceUrl"`
	SubscriptionRequired bool   `json:"subscriptionRequired"`
}

func (c *Client) ListAPIs() ([]API, error) {
	body, status, err := c.request("GET", c.apimPath("/apis"), nil)
	if err != nil {
		return nil, err
	}

	if status != 200 {
		return nil, fmt.Errorf("failed to list APIs (status %d): %s", status, string(body))
	}

	var result struct {
		Value []struct {
			Name       string `json:"name"`
			Properties struct {
				DisplayName string `json:"displayName"`
				Path        string `json:"path"`
				ServiceURL  string `json:"serviceUrl"`
			} `json:"properties"`
		} `json:"value"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	apis := make([]API, len(result.Value))
	for i, v := range result.Value {
		parts := strings.Split(v.Name, "/")
		apiID := parts[len(parts)-1]
		apis[i] = API{
			Name:        apiID,
			DisplayName: v.Properties.DisplayName,
			Path:        v.Properties.Path,
			ServiceURL:  v.Properties.ServiceURL,
		}
	}

	return apis, nil
}

func (c *Client) GetAPI(apiID string) (*API, error) {
	body, status, err := c.request("GET", c.apimPath("/apis/"+apiID), nil)
	if err != nil {
		return nil, err
	}

	if status == 404 {
		return nil, fmt.Errorf("API '%s' not found", apiID)
	}
	if status != 200 {
		return nil, fmt.Errorf("failed to get API (status %d): %s", status, string(body))
	}

	var result struct {
		Name       string `json:"name"`
		Properties struct {
			DisplayName          string `json:"displayName"`
			Path                 string `json:"path"`
			ServiceURL           string `json:"serviceUrl"`
			SubscriptionRequired *bool  `json:"subscriptionRequired"`
		} `json:"properties"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	subRequired := true
	if result.Properties.SubscriptionRequired != nil {
		subRequired = *result.Properties.SubscriptionRequired
	}

	return &API{
		Name:                 apiID,
		DisplayName:          result.Properties.DisplayName,
		Path:                 result.Properties.Path,
		ServiceURL:           result.Properties.ServiceURL,
		SubscriptionRequired: subRequired,
	}, nil
}

type NamedValue struct {
	Name   string
	Value  string
	Secret bool
}

func (c *Client) GetNamedValue(name string) (string, error) {
	nv, err := c.GetNamedValueInfo(name)
	if err != nil || nv == nil {
		return "", err
	}
	return nv.Value, nil
}

func (c *Client) GetNamedValueInfo(name string) (*NamedValue, error) {
	body, status, err := c.request("GET", c.apimPath("/namedValues/"+name), nil)
	if err != nil {
		return nil, err
	}

	if status == 404 {
		return nil, nil
	}
	if status != 200 {
		return nil, fmt.Errorf("failed to get named value (status %d): %s", status, string(body))
	}

	var result struct {
		Properties struct {
			DisplayName string `json:"displayName"`
			Value       string `json:"value"`
			Secret      bool   `json:"secret"`
		} `json:"properties"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	value := result.Properties.Value
	if result.Properties.Secret {
		value = "[secret]"
	}
	return &NamedValue{
		Name:   name,
		Value:  value,
		Secret: result.Properties.Secret,
	}, nil
}

func (c *Client) CreateOrUpdateNamedValue(name, displayName, value string, secret bool) error {
	payload := map[string]any{
		"properties": map[string]any{
			"displayName": displayName,
			"value":       value,
			"secret":      secret,
		},
	}

	body, status, headers, err := c.requestWithHeaders("PUT", c.apimPath("/namedValues/"+name), payload)
	if err != nil {
		return err
	}

	if status != 200 && status != 201 && status != 202 {
		return fmt.Errorf("failed to create/update named value (status %d): %s", status, string(body))
	}

	if status == 202 {
		if err := c.waitForAsyncOperation(headers, "named value", name); err != nil {
			return err
		}
	}

	return nil
}

type PolicyFragment struct {
	Name        string
	Description string
}

func (c *Client) GetPolicyFragment(name string) (bool, error) {
	_, exists, err := c.GetPolicyFragmentContent(name)
	return exists, err
}

func (c *Client) GetPolicyFragmentContent(name string) (string, bool, error) {
	query := url.Values{}
	query.Set("format", "rawxml")
	body, status, _, err := c.requestWithHeadersAndQuery("GET", c.apimPath("/policyFragments/"+name), query, nil)
	if err != nil {
		return "", false, err
	}

	if status == 404 {
		return "", false, nil
	}
	if status != 200 {
		return "", false, fmt.Errorf("failed to get policy fragment '%s' (status %d): %s", name, status, string(body))
	}

	var result struct {
		Properties struct {
			Value string `json:"value"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", false, fmt.Errorf("failed to parse policy fragment '%s': %w", name, err)
	}
	return result.Properties.Value, true, nil
}

func (c *Client) CreateOrUpdatePolicyFragment(name, xmlContent, description string) error {
	properties := map[string]any{
		"format": "xml",
		"value":  xmlContent,
	}
	if description != "" {
		properties["description"] = description
	}
	payload := map[string]any{
		"properties": properties,
	}

	body, status, headers, err := c.requestWithHeaders("PUT", c.apimPath("/policyFragments/"+name), payload)
	if err != nil {
		return err
	}

	if status != 200 && status != 201 && status != 202 {
		return fmt.Errorf("failed to create/update policy fragment '%s' (status %d): %s", name, status, string(body))
	}

	if status == 202 {
		if err := c.waitForAsyncOperation(headers, "policy fragment", name); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) SetAPIPolicy(apiID, xmlContent string) error {
	payload := map[string]any{
		"properties": map[string]any{
			"format": "xml",
			"value":  xmlContent,
		},
	}

	body, status, headers, err := c.requestWithHeaders("PUT", c.apimPath("/apis/"+apiID+"/policies/policy"), payload)
	if err != nil {
		return err
	}

	if status != 200 && status != 201 && status != 202 {
		return fmt.Errorf("failed to set API policy (status %d): %s", status, string(body))
	}

	if status == 202 {
		if err := c.waitForAsyncOperation(headers, "api policy", apiID); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) waitForAsyncOperation(headers http.Header, resourceType, name string) error {
	asyncURL := headers.Get("Azure-AsyncOperation")
	if asyncURL == "" {
		asyncURL = headers.Get("Location")
	}
	if asyncURL == "" {
		time.Sleep(2 * time.Second)
		return nil
	}

	const (
		maxAttempts = 30
		delay       = 2 * time.Second
	)

	for range maxAttempts {
		status, body, err := c.getAsyncStatus(asyncURL)
		if err != nil {
			return err
		}
		switch strings.ToLower(status) {
		case "succeeded":
			return nil
		case "failed", "canceled":
			return fmt.Errorf("%s '%s' failed: %s", resourceType, name, body)
		}
		time.Sleep(delay)
	}

	return fmt.Errorf("%s '%s' provisioning did not complete in time", resourceType, name)
}

func (c *Client) getAsyncStatus(asyncURL string) (string, string, error) {
	req, err := http.NewRequest("GET", asyncURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create async status request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("async status request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read async status response: %w", err)
	}

	if c.verbose {
		c.logResponse("GET", asyncURL, resp.StatusCode, resp.Header, body)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", string(body), fmt.Errorf("async status request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string `json:"status"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "unknown", string(body), nil
	}
	if result.Status == "" {
		result.Status = "unknown"
	}
	if result.Error.Message != "" {
		return result.Status, fmt.Sprintf("%s: %s", result.Error.Code, result.Error.Message), nil
	}
	return result.Status, string(body), nil
}

func (c *Client) GetGatewayURL() (string, error) {
	body, status, err := c.request("GET", c.apimPath(""), nil)
	if err != nil {
		return "", err
	}

	if status != 200 {
		return "", fmt.Errorf("failed to get APIM service (status %d): %s", status, string(body))
	}

	var result struct {
		Properties struct {
			GatewayURL string `json:"gatewayUrl"`
		} `json:"properties"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Properties.GatewayURL, nil
}

func (c *Client) GetAPIPolicy(apiID string) (string, error) {
	body, status, err := c.request("GET", c.apimPath("/apis/"+apiID+"/policies/policy"), nil)
	if err != nil {
		return "", err
	}

	if status == 404 {
		return "", nil
	}
	if status != 200 {
		return "", fmt.Errorf("failed to get API policy (status %d): %s", status, string(body))
	}

	var result struct {
		Properties struct {
			Value string `json:"value"`
		} `json:"properties"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Properties.Value, nil
}
