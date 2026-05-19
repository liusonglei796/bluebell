package jwt

import (
	"bluebell/internal/config"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GetGitHubConfig 获取 GitHub OAuth2 配置
func GetGitHubConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GitHub.ClientID,
		ClientSecret: cfg.GitHub.ClientSecret,
		RedirectURL:  cfg.GitHub.RedirectURL,
		Scopes:       []string{"user:email", "read:user"},
		Endpoint:     github.Endpoint,
	}
}

// GetGitHubUserInfo 获取 GitHub 用户信息
func GetGitHubUserInfo(token *oauth2.Token) (map[string]interface{}, error) {
	client := http.DefaultClient
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status: %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Fetch email separately if it's not public
	if userInfo["email"] == nil {
		req, _ = http.NewRequest("GET", "https://api.github.com/user/emails", nil)
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		resp, err = client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			var emails []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&emails); err == nil {
				for _, e := range emails {
					if primary, ok := e["primary"].(bool); ok && primary {
						userInfo["email"] = e["email"]
						break
					}
				}
			}
		}
	}

	return userInfo, nil
}
