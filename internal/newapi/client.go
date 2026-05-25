package newapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client 封装对 new-api 的 HTTP 调用
type Client struct {
	BaseURL     string
	AdminKey    string
	AdminUserID int // new-api 中 admin key 对应的用户 ID，用于 New-Api-User 头
	HTTPClient  *http.Client
}

func NewClient(baseURL, adminKey string, adminUserID int) *Client {
	return &Client{
		BaseURL:     baseURL,
		AdminKey:    adminKey,
		AdminUserID: adminUserID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterUser 在 new-api 中注册一个内部用户
func (c *Client) RegisterUser(email, password string) (int, error) {
	body := map[string]string{
		"username":     email,
		"password":     password,
		"display_name": email,
	}
	return c.doRegister(body)
}

// RegisterUserWithUsername 在 new-api 中注册一个带用户名的内部用户
func (c *Client) RegisterUserWithUsername(username, email, password string) (int, error) {
	body := map[string]string{
		"username":     username,
		"password":     password,
		"display_name": username,
	}
	return c.doRegister(body)
}

func (c *Client) doRegister(body map[string]string) (int, error) {
	data, _ := json.Marshal(body)
	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/api/user/register",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return 0, fmt.Errorf("register user: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode register response: %w", err)
	}
	if !result.Success {
		return 0, fmt.Errorf("new-api register failed: %s", result.Message)
	}
	return result.Data.ID, nil
}

// GetUserToken 获取用户在 new-api 中的 API Token（需 admin 权限）
func (c *Client) GetUserToken(userID int) (string, error) {
	url := fmt.Sprintf("%s/api/user/token?id=%d", c.BaseURL, userID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get user token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if !result.Success {
		return "", fmt.Errorf("get token failed: %s", result.Message)
	}
	return result.Data, nil
}

// GetUserInfo 获取 new-api 用户信息（需 admin 权限）
func (c *Client) GetUserInfo(userID int) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/user/%d", c.BaseURL, userID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)
	// new-api 的 AdminAuth 中间件要求 New-Api-User 头匹配 admin 用户自身 ID
	req.Header.Set("New-Api-User", fmt.Sprintf("%d", c.AdminUserID))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool     `json:"success"`
		Message string   `json:"message"`
		Data    UserInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("get user info failed: %s", result.Message)
	}
	return &result.Data, nil
}

type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Quota    int64  `json:"quota"`
}

// PricingResponse new-api 定价接口返回结构
type PricingResponse struct {
	Success    bool                     `json:"success"`
	Message    string                   `json:"message"`
	Data       []PricingItem            `json:"data"`
	Vendors    []PricingVendor          `json:"vendors"`
	GroupRatio map[string]float64       `json:"group_ratio"`
	AutoGroups []string                 `json:"auto_groups"`
}

type PricingItem struct {
	ModelName              string   `json:"model_name"`
	VendorID               int      `json:"vendor_id"`
	QuotaType              int      `json:"quota_type"`
	ModelRatio             float64  `json:"model_ratio"`
	ModelPrice             float64  `json:"model_price"`
	OwnerBy                string   `json:"owner_by"`
	CompletionRatio        float64  `json:"completion_ratio"`
	EnableGroups           []string `json:"enable_groups"`
	SupportedEndpointTypes []string `json:"supported_endpoint_types"`
}

type PricingVendor struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// GetPricing 获取 new-api 的模型定价信息（公开接口，无需鉴权）
func (c *Client) GetPricing() (*PricingResponse, error) {
	url := c.BaseURL + "/api/pricing"
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get pricing: %w", err)
	}
	defer resp.Body.Close()

	var result PricingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode pricing response: %w", err)
	}
	return &result, nil
}

// AdminCreateToken 使用管理员权限为指定用户创建 API Token（返回完整 key 和 token ID）
func (c *Client) AdminCreateToken(userID int, tokenName string) (int, string, error) {
	return c.AdminCreateTokenWithQuota(userID, tokenName, nil, nil)
}

// AdminCreateTokenWithQuota 使用管理员权限创建带配额的 API Token
func (c *Client) AdminCreateTokenWithQuota(userID int, tokenName string, remainQuota *int, unlimited *bool) (int, string, error) {
	body := map[string]interface{}{
		"user_id": userID,
		"name":    tokenName,
	}
	if remainQuota != nil {
		body["remain_quota"] = *remainQuota
	}
	if unlimited != nil {
		body["unlimited_quota"] = *unlimited
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/admin/token/create",
		bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("admin create token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			ID  int    `json:"id"`
			Key string `json:"key"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", fmt.Errorf("decode create token response: %w", err)
	}
	if !result.Success {
		return 0, "", fmt.Errorf("create token failed: %s", result.Message)
	}
	return result.Data.ID, result.Data.Key, nil
}

// AdminUpdateTokenStatus 使用管理员权限更新 token 状态（1=启用, 2=禁用）
func (c *Client) AdminUpdateTokenStatus(tokenID int, status int) error {
	body := map[string]interface{}{
		"token_id": tokenID,
		"status":   status,
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/admin/token/status",
		bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("admin update token status: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode update token status response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("update token status failed: %s", result.Message)
	}
	return nil
}

// AdminUpdateTokenQuota 使用管理员权限更新 token 配额
func (c *Client) AdminUpdateTokenQuota(tokenID int, remainQuota *int, unlimited *bool) error {
	body := map[string]interface{}{
		"token_id": tokenID,
	}
	if remainQuota != nil {
		body["remain_quota"] = *remainQuota
	}
	if unlimited != nil {
		body["unlimited_quota"] = *unlimited
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/admin/token/quota",
		bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("admin update token quota: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode update token quota response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("update token quota failed: %s", result.Message)
	}
	return nil
}

// TokenInfo new-api 返回的 token 信息
type TokenInfo struct {
	ID             int    `json:"id"`
	UserID         int    `json:"user_id"`
	Name           string `json:"name"`
	Status         int    `json:"status"`
	RemainQuota    int    `json:"remain_quota"`
	UnlimitedQuota bool   `json:"unlimited_quota"`
	UsedQuota      int    `json:"used_quota"`
}

// AdminGetTokenInfo 使用管理员权限查询 token 信息
func (c *Client) AdminGetTokenInfo(tokenID int) (*TokenInfo, error) {
	body := map[string]interface{}{
		"token_id": tokenID,
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/admin/token/info",
		bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("admin get token info: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool      `json:"success"`
		Message string    `json:"message"`
		Data    TokenInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode get token info response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("get token info failed: %s", result.Message)
	}
	return &result.Data, nil
}

// AdminDeleteToken 使用管理员权限删除指定 token
func (c *Client) AdminDeleteToken(tokenID int) error {
	body := map[string]interface{}{
		"token_id": tokenID,
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/admin/token/delete",
		bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("admin delete token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode delete token response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("delete token failed: %s", result.Message)
	}
	return nil
}

// StatusResponse new-api /api/status 返回
type StatusResponse struct {
	QuotaPerUnit float64 `json:"quota_per_unit"`
}

// GetStatus 获取 new-api 的运行状态信息（公开接口，无需鉴权）
func (c *Client) GetStatus() (*StatusResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/status")
	if err != nil {
		return nil, fmt.Errorf("get status: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool          `json:"success"`
		Data    StatusResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode status response: %w", err)
	}
	return &result.Data, nil
}

// LogItem new-api 调用日志条目
type LogItem struct {
	ID               int     `json:"id"`
	UserID           int     `json:"user_id"`
	CreatedAt        int64   `json:"created_at"`
	Type             int     `json:"type"`
	Content          string  `json:"content"`
	Username         string  `json:"username"`
	TokenName        string  `json:"token_name"`
	ModelName        string  `json:"model_name"`
	Quota            int     `json:"quota"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	UseTime          float64 `json:"use_time"`
	IsStream         bool    `json:"is_stream"`
	Channel          int     `json:"channel"`
	ChannelName      string  `json:"channel_name"`
	TokenID          int     `json:"token_id"`
	Group            string  `json:"group"`
	IP               string  `json:"ip"`
	RequestID        string  `json:"request_id"`
	Other            string  `json:"other"`
}

// GetLogsByToken 使用 API Key 查询该 token 的调用日志
func (c *Client) GetLogsByToken(token string) ([]LogItem, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/api/log/token", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get logs by token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool      `json:"success"`
		Message string    `json:"message"`
		Data    []LogItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode logs response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("get logs failed: %s", result.Message)
	}
	return result.Data, nil
}
