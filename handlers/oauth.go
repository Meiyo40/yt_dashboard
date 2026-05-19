package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"yt_dashboard/db"
)

// generateState creates a secure random string for CSRF mitigation
func generateState() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "static_fallback_state_123"
	}
	return hex.EncodeToString(b)
}

// OAuthLoginHandler redirects the user to Google's OAuth consent screen.
func OAuthLoginHandler(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("YT_CLIENT_ID")
	redirectURI := os.Getenv("YT_REDIRECT_URI")

	if clientID == "" || redirectURI == "" {
		log.Println("OAuth Configuration Error: YT_CLIENT_ID or YT_REDIRECT_URI is not set")
		http.Error(w, "OAuth application is not configured. Please set YT_CLIENT_ID and YT_REDIRECT_URI in the environment.", http.StatusInternalServerError)
		return
	}

	state := generateState()

	// Store state in a short-lived HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(5 * time.Minute),
		HttpOnly: true,
		Secure:   r.URL.Scheme == "https" || strings.HasPrefix(r.Host, "localhost:"), // Secure cookie on HTTPS or localhost
		SameSite: http.SameSiteLaxMode,
	})

	// Construct Google OAuth URL
	u, err := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	if err != nil {
		http.Error(w, "Failed to parse oauth url", http.StatusInternalServerError)
		return
	}

	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "https://www.googleapis.com/auth/youtube.readonly https://www.googleapis.com/auth/youtube.force-ssl")
	q.Set("access_type", "offline")
	q.Set("prompt", "consent") // Ensures refresh_token is always returned
	q.Set("state", state)

	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}

type tokenExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// OAuthCallbackHandler handles the OAuth2 callback redirect from Google.
func OAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stateQuery := r.URL.Query().Get("state")
	codeQuery := r.URL.Query().Get("code")

	// 1. Verify CSRF State
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		log.Printf("OAuth Callback: Missing state cookie: %v", err)
		http.Error(w, "Missing state cookie. Please try logging in again.", http.StatusBadRequest)
		return
	}

	// Delete state cookie immediately
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	if stateQuery != cookie.Value {
		log.Printf("OAuth Callback: State mismatch. Query state: %s, Cookie state: %s", stateQuery, cookie.Value)
		http.Error(w, "CSRF validation failed: State parameter mismatch.", http.StatusBadRequest)
		return
	}

	if codeQuery == "" {
		errReason := r.URL.Query().Get("error")
		log.Printf("OAuth Callback: Google returned error: %s", errReason)
		http.Error(w, fmt.Sprintf("OAuth error: %s", errReason), http.StatusBadRequest)
		return
	}

	// 2. Load Client Credentials
	clientID := os.Getenv("YT_CLIENT_ID")
	clientSecret := os.Getenv("YT_CLIENT_SECRET")
	redirectURI := os.Getenv("YT_REDIRECT_URI")

	if clientID == "" || clientSecret == "" || redirectURI == "" {
		log.Println("OAuth Callback Error: Missing YT credentials in environment")
		http.Error(w, "Server configuration error: missing YouTube client details.", http.StatusInternalServerError)
		return
	}

	// 3. Exchange Authorization Code for Access & Refresh Tokens
	val := url.Values{}
	val.Set("client_id", clientID)
	val.Set("client_secret", clientSecret)
	val.Set("code", codeQuery)
	val.Set("redirect_uri", redirectURI)
	val.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(val.Encode()))
	if err != nil {
		log.Printf("OAuth Callback: Error creating token request: %v", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("OAuth Callback: Error sending token exchange request: %v", err)
		http.Error(w, "Failed to connect to Google OAuth service.", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errData map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errData)
		log.Printf("OAuth Callback: Google token exchange returned status %d. Response: %v", resp.StatusCode, errData)
		http.Error(w, "Failed to exchange authorization code for tokens.", http.StatusBadRequest)
		return
	}

	var res tokenExchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Printf("OAuth Callback: Error decoding token response: %v", err)
		http.Error(w, "Failed to parse credentials response.", http.StatusInternalServerError)
		return
	}

	if res.RefreshToken == "" {
		log.Println("OAuth Callback Warning: Google did not return a refresh token. The application may fail to refresh the connection in 1 hour.")
		// Wait, if SQLite schema requires refresh_token to be NOT NULL, this will crash.
		// However, because we forced `prompt=consent` and `access_type=offline`, a refresh_token
		// MUST be provided. If not, we should notify the user.
		http.Error(w, "Failed to obtain refresh token. Please disconnect the app in your Google account settings and log in again.", http.StatusInternalServerError)
		return
	}

	// 4. Save to Database
	token := &db.OAuthToken{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		TokenType:    res.TokenType,
		Expiry:       time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
	}

	if err := db.SaveToken(ctx, token); err != nil {
		log.Printf("OAuth Callback: Failed to save tokens to database: %v", err)
		http.Error(w, "Database error: failed to store connection.", http.StatusInternalServerError)
		return
	}

	log.Println("YouTube account successfully connected and OAuth token stored.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// OAuthLogoutHandler deletes the connection token from the database.
func OAuthLogoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := db.DeleteToken(ctx); err != nil {
		log.Printf("OAuth Logout: Failed to delete token from database: %v", err)
		http.Error(w, "Failed to disconnect account.", http.StatusInternalServerError)
		return
	}

	log.Println("YouTube account successfully disconnected.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
