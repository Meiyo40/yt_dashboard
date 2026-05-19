package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestTokensCRUD(t *testing.T) {
	// Initialize a temporary database for testing
	testDBPath := "test_yt_dashboard.db"
	defer os.Remove(testDBPath)

	database, err := InitDB(testDBPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// 1. GetToken on empty DB should fail
	_, err = GetToken(ctx)
	if err == nil {
		t.Error("expected error getting token from empty database, got nil")
	}

	// 2. Save a new token
	now := time.Now().Truncate(time.Second) // SQLite RFC3339 storage might truncate fractional seconds depending on formatting
	token := &OAuthToken{
		AccessToken:  "access-123",
		RefreshToken: "refresh-123",
		TokenType:    "Bearer",
		Expiry:       now.Add(1 * time.Hour),
	}

	err = SaveToken(ctx, token)
	if err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// 3. GetToken should succeed and match
	retrieved, err := GetToken(ctx)
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}

	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("expected access token %q, got %q", token.AccessToken, retrieved.AccessToken)
	}
	if retrieved.RefreshToken != token.RefreshToken {
		t.Errorf("expected refresh token %q, got %q", token.RefreshToken, retrieved.RefreshToken)
	}
	if retrieved.TokenType != token.TokenType {
		t.Errorf("expected token type %q, got %q", token.TokenType, retrieved.TokenType)
	}
	// Note: Compare times using Equal to handle locations and monotonic differences
	if !retrieved.Expiry.Equal(token.Expiry) {
		t.Errorf("expected expiry %v, got %v", token.Expiry, retrieved.Expiry)
	}

	// 4. Update the token with empty refresh token (simulating an OAuth provider refresh that didn't rotate refresh token)
	updatedToken := &OAuthToken{
		AccessToken:  "access-456",
		RefreshToken: "",
		TokenType:    "Bearer",
		Expiry:       now.Add(2 * time.Hour),
	}
	err = SaveToken(ctx, updatedToken)
	if err != nil {
		t.Fatalf("failed to save updated token: %v", err)
	}

	retrieved, err = GetToken(ctx)
	if err != nil {
		t.Fatalf("failed to get updated token: %v", err)
	}
	// Access token and expiry should be updated, but refresh token should NOT be overwritten (since it was empty)
	if retrieved.AccessToken != "access-456" {
		t.Errorf("expected updated access token %q, got %q", "access-456", retrieved.AccessToken)
	}
	if retrieved.RefreshToken != "refresh-123" {
		t.Errorf("expected refresh token to be preserved as %q, got %q", "refresh-123", retrieved.RefreshToken)
	}

	// 5. Delete token
	err = DeleteToken(ctx)
	if err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	_, err = GetToken(ctx)
	if err == nil {
		t.Error("expected error getting token after deletion, got nil")
	}
}
