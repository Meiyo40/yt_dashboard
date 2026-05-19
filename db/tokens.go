package db

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// SaveToken saves or updates the oauth token in the database.
func SaveToken(ctx context.Context, token *OAuthToken) error {
	query := `
		INSERT INTO oauth_tokens (id, access_token, refresh_token, token_type, expiry)
		VALUES ('current', ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = CASE 
				WHEN excluded.refresh_token <> '' THEN excluded.refresh_token 
				ELSE refresh_token 
			END,
			token_type = excluded.token_type,
			expiry = excluded.expiry
	`
	expiryStr := token.Expiry.Format(time.RFC3339)
	_, err := DB.ExecContext(ctx, query, token.AccessToken, token.RefreshToken, token.TokenType, expiryStr)
	return err
}

// GetToken retrieves the stored oauth token from the database.
// Returns sql.ErrNoRows if no token is found.
func GetToken(ctx context.Context) (*OAuthToken, error) {
	query := `
		SELECT access_token, refresh_token, token_type, expiry
		FROM oauth_tokens
		WHERE id = 'current'
	`
	var token OAuthToken
	var expiryStr string
	err := DB.QueryRowContext(ctx, query).Scan(&token.AccessToken, &token.RefreshToken, &token.TokenType, &expiryStr)
	if err != nil {
		return nil, err
	}
	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token expiry: %w", err)
	}
	token.Expiry = expiry
	return &token, nil
}

// DeleteToken deletes the oauth token from the database.
func DeleteToken(ctx context.Context) error {
	_, err := DB.ExecContext(ctx, "DELETE FROM oauth_tokens WHERE id = 'current'")
	return err
}

// GetValidToken returns a valid OAuth token. If the token is expired or expiring in less than 5 minutes,
// it refreshes the token automatically and stores the refreshed token in the database.
func GetValidToken(ctx context.Context) (*OAuthToken, error) {
	token, err := GetToken(ctx)
	if err != nil {
		return nil, err
	}

	// If the token is valid for at least another 5 minutes, return it
	if time.Now().Add(5 * time.Minute).Before(token.Expiry) {
		return token, nil
	}

	// Token is expired or expiring soon, let's refresh it
	clientID := os.Getenv("YT_CLIENT_ID")
	clientSecret := os.Getenv("YT_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("missing YT_CLIENT_ID or YT_CLIENT_SECRET environment variables")
	}

	refreshedToken, err := refreshOAuthToken(ctx, token.RefreshToken, clientID, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if err := SaveToken(ctx, refreshedToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	return refreshedToken, nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

func refreshOAuthToken(ctx context.Context, refreshToken, clientID, clientSecret string) (*OAuthToken, error) {
	val := url.Values{}
	val.Set("client_id", clientID)
	val.Set("client_secret", clientSecret)
	val.Set("refresh_token", refreshToken)
	val.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(val.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh token request failed with status: %d", resp.StatusCode)
	}

	var res tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	newRefreshToken := refreshToken
	if res.RefreshToken != "" {
		newRefreshToken = res.RefreshToken
	}

	return &OAuthToken{
		AccessToken:  res.AccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    res.TokenType,
		Expiry:       time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
	}, nil
}
