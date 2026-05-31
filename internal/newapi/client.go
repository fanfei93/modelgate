package newapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client 封装对内部模型服务的 HTTP 调用
type Client struct {
	BaseURL     string
	AdminKey    string
	AdminUserID int
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

// RegisterUser 在内部服务中注册一个用户
func (c *Client) RegisterUser(email, password string) (int, error) {
	body := map[string]string{
		"username":     email,
		"password":     password,
		"display_name": email,
	}
	return c.doRegister(body)
}

// RegisterUserWithUsername 在内部服务中注册一个带用户名的用户
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
		return 0, fmt.Errorf("注册用户失败")
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
		return 0, fmt.Errorf("注册用户失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 注册用户失败: %s", result.Message)
		return 0, fmt.Errorf("注册用户失败")
	}
	return result.Data.ID, nil
}

// GetUserToken 获取用户的 API Token（需 admin 权限）
func (c *Client) GetUserToken(userID int) (string, error) {
	url := fmt.Sprintf("%s/api/user/token?id=%d", c.BaseURL, userID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取令牌失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("获取令牌失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 获取令牌失败: %s", result.Message)
		return "", fmt.Errorf("获取令牌失败")
	}
	return result.Data, nil
}

// UserLogin 通过用户名密码登录，验证凭据是否有效
// new-api 的 login 接口成功时 data 为对象而非字符串，这里只关心 success 状态
func (c *Client) UserLogin(username, password string) error {
	body := map[string]string{
		"username": username,
		"password": password,
	}
	data, _ := json.Marshal(body)
	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/api/user/login",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf("登录验证失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("登录验证失败")
	}
	if !result.Success {
		return fmt.Errorf("登录验证失败")
	}
	return nil
}

// GetUserInfo 获取用户信息（需 admin 权限）
func (c *Client) GetUserInfo(userID int) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/user/%d", c.BaseURL, userID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)
	req.Header.Set("New-Api-User", fmt.Sprintf("%d", c.AdminUserID))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool     `json:"success"`
		Message string   `json:"message"`
		Data    UserInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("获取用户信息失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 获取用户信息失败 (userID=%d): %s", userID, result.Message)
		return nil, fmt.Errorf("获取用户信息失败")
	}
	return &result.Data, nil
}

type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Quota    int64  `json:"quota"`
}

// PricingResponse 模型定价接口返回结构
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

// GetPricing 获取模型定价信息（公开接口，无需鉴权）
func (c *Client) GetPricing() (*PricingResponse, error) {
	url := c.BaseURL + "/api/pricing"
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取模型定价失败")
	}
	defer resp.Body.Close()

	var result PricingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("获取模型定价失败")
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
		return 0, "", fmt.Errorf("创建令牌失败")
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
		return 0, "", fmt.Errorf("创建令牌失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 创建令牌失败: %s", result.Message)
		return 0, "", fmt.Errorf("创建令牌失败")
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
		return fmt.Errorf("更新令牌状态失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("更新令牌状态失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 更新令牌状态失败 (tokenID=%d): %s", tokenID, result.Message)
		return fmt.Errorf("更新令牌状态失败")
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
		return fmt.Errorf("更新令牌配额失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("更新令牌配额失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 更新令牌配额失败 (tokenID=%d): %s", tokenID, result.Message)
		return fmt.Errorf("更新令牌配额失败")
	}
	return nil
}

// TokenInfo 返回的 token 信息
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
		return nil, fmt.Errorf("获取令牌信息失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool      `json:"success"`
		Message string    `json:"message"`
		Data    TokenInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("获取令牌信息失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 获取令牌信息失败 (tokenID=%d): %s", tokenID, result.Message)
		return nil, fmt.Errorf("获取令牌信息失败")
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
		return fmt.Errorf("删除令牌失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("删除令牌失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 删除令牌失败 (tokenID=%d): %s", tokenID, result.Message)
		return fmt.Errorf("删除令牌失败")
	}
	return nil
}

// StatusResponse /api/status 返回
type StatusResponse struct {
	QuotaPerUnit float64 `json:"quota_per_unit"`
}

// GetStatus 获取运行状态信息（公开接口，无需鉴权）
func (c *Client) GetStatus() (*StatusResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/status")
	if err != nil {
		return nil, fmt.Errorf("获取服务状态失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool          `json:"success"`
		Data    StatusResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("获取服务状态失败")
	}
	return &result.Data, nil
}

// LogItem 调用日志条目
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

// AdminUpdateUserQuota 使用管理员权限覆盖用户的总配额（user.Quota）
func (c *Client) AdminUpdateUserQuota(userID int, quota int) error {
	body := map[string]interface{}{
		"id":     userID,
		"action": "add_quota",
		"value":  quota,
		"mode":   "override",
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/user/manage",
		bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)
	req.Header.Set("New-Api-User", fmt.Sprintf("%d", c.AdminUserID))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("更新用户配额失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("更新用户配额失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 更新用户配额失败 (userID=%d): %s", userID, result.Message)
		return fmt.Errorf("更新用户配额失败")
	}
	return nil
}

// GetLogsByToken 使用 API Key 查询该 token 的调用日志
func (c *Client) GetLogsByToken(token string) ([]LogItem, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/api/log/token", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取日志失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool      `json:"success"`
		Message string    `json:"message"`
		Data    []LogItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("获取日志失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 获取日志失败: %s", result.Message)
		return nil, fmt.Errorf("获取日志失败")
	}
	return result.Data, nil
}

// LogsQuery 调用日志查询参数
type LogsQuery struct {
	Username       string
	TokenName      string
	TokenID        int
	ModelName      string
	StartTimestamp int64
	EndTimestamp   int64
	Page           int
	PageSize       int
}

// PaginatedLogs 分页日志结果
type PaginatedLogs struct {
	Items    []LogItem `json:"items"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

// GetLogsByUserID 使用管理员权限查询指定用户的所有调用日志
func (c *Client) GetLogsByUserID(q LogsQuery) (*PaginatedLogs, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}

	// 如果指定了 TokenID，获取原始 token_name 用于筛选
	if q.TokenID > 0 && q.TokenName == "" {
		tokenInfo, err := c.AdminGetTokenInfo(q.TokenID)
		if err == nil && tokenInfo != nil {
			q.TokenName = tokenInfo.Name
		}
	}

	params := url.Values{}
	params.Set("username", q.Username)
	params.Set("p", strconv.Itoa(q.Page))
	params.Set("page_size", strconv.Itoa(q.PageSize))
	if q.TokenName != "" {
		params.Set("token_name", q.TokenName)
	}
	if q.ModelName != "" {
		params.Set("model_name", q.ModelName)
	}
	if q.StartTimestamp > 0 {
		params.Set("start_timestamp", strconv.FormatInt(q.StartTimestamp, 10))
	}
	if q.EndTimestamp > 0 {
		params.Set("end_timestamp", strconv.FormatInt(q.EndTimestamp, 10))
	}

	reqURL := fmt.Sprintf("%s/api/log/?%s", c.BaseURL, params.Encode())
	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)
	req.Header.Set("New-Api-User", fmt.Sprintf("%d", c.AdminUserID))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取日志失败")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Items    []LogItem `json:"items"`
			Total    int       `json:"total"`
			Page     int       `json:"page"`
			PageSize int       `json:"page_size"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("获取日志失败")
	}
	if !result.Success {
		log.Printf("[ERROR] 获取日志失败: %s", result.Message)
		return nil, fmt.Errorf("获取日志失败")
	}

	return &PaginatedLogs{
		Items:    result.Data.Items,
		Total:    result.Data.Total,
		Page:     result.Data.Page,
		PageSize: result.Data.PageSize,
	}, nil
}
