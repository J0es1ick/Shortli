package repository_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/app/routes"
	"github.com/J0es1ick/shortli/internal/app/tasks"
	"github.com/J0es1ick/shortli/internal/config"
	"github.com/J0es1ick/shortli/internal/database"
	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/repository"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

func TestSecurityLifecycleWithPostgres(t *testing.T) {
	databaseURL := os.Getenv("SHORTLI_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SHORTLI_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sqlx.ConnectContext(ctx, "pgx", databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE user_info, url_info, session_info, api_key, abuse_report, blocked_domain, click_event RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset database: %v", err)
	}

	users := repository.NewUserRepository(db)
	sessions := repository.NewSessionRepository(db)
	urls := repository.NewUrlRepository(db)
	first := &models.User{Email: "first@example.com", PasswordHash: "hash", Role: models.RoleOwner}
	if err := users.BootstrapAdmin(ctx, first); err != nil {
		t.Fatalf("bootstrap first admin: %v", err)
	}
	if err := users.BootstrapAdmin(ctx, &models.User{Email: "other@example.com", PasswordHash: "hash", Role: models.RoleOwner}); !errors.Is(err, repository.ErrAdminAlreadyExists) {
		t.Fatalf("second bootstrap error = %v", err)
	}
	if _, err := users.DeleteUser(ctx, first.ID); !errors.Is(err, repository.ErrLastOwner) {
		t.Fatalf("delete last admin error = %v", err)
	}

	second := &models.User{Email: "second@example.com", PasswordHash: "hash"}
	if err := users.SaveUser(ctx, second); err != nil {
		t.Fatalf("save second user: %v", err)
	}
	second.Role = models.RoleOwner
	if err := users.UpdateUser(ctx, second); err != nil {
		t.Fatalf("promote second admin: %v", err)
	}
	link := &models.URL{
		OriginalURL: "https://example.com", ShortCode: "first-link", UserID: &first.ID,
		CreatedAt: time.Now().UTC(), IsActive: true,
	}
	if _, err := urls.SaveUrl(ctx, link); err != nil {
		t.Fatalf("save owned link: %v", err)
	}
	if _, err := users.DeleteUser(ctx, first.ID); err != nil {
		t.Fatalf("delete admin with replacement: %v", err)
	}
	var linkCount int
	if err := db.GetContext(ctx, &linkCount, `SELECT COUNT(*) FROM url_info WHERE short_code = 'first-link'`); err != nil || linkCount != 0 {
		t.Fatalf("owned link count = %d, err = %v", linkCount, err)
	}

	session := &models.Session{ID: "raw-session-token", UserID: second.ID, ExpiresAt: time.Now().Add(time.Hour)}
	if err := sessions.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	var storedID string
	if err := db.GetContext(ctx, &storedID, `SELECT session_id FROM session_info WHERE user_id = $1`, second.ID); err != nil {
		t.Fatalf("load stored session: %v", err)
	}
	if storedID == session.ID || len(storedID) != 64 {
		t.Fatalf("session token was not hashed")
	}
	if _, err := sessions.GetSessionByID(ctx, session.ID); err != nil {
		t.Fatalf("authenticate hashed session: %v", err)
	}

	apiKeys := repository.NewAPIKeyRepository(db)
	abuse := repository.NewAbuseRepository(db)
	clickRecorder, err := tasks.NewClickRecorder(urls, t.TempDir(), 1)
	if err != nil {
		t.Fatalf("create click recorder: %v", err)
	}
	defer func() {
		closeContext, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		if err := clickRecorder.Close(closeContext); err != nil {
			t.Errorf("close click recorder: %v", err)
		}
	}()
	clientIP := middleware.NewClientIPResolver("")
	handler := routes.SetupRoutes(
		&config.Config{
			PublicBaseURL: "http://shortli.test", FrontendOrigin: "http://shortli.test",
			AnalyticsSalt: "0123456789abcdef0123456789abcdef", RequestTimeout: 5,
		},
		urls, users, sessions, apiKeys, abuse, clickRecorder,
		middleware.NewMetricsRegistry(clientIP), clientIP,
	)
	server := httptest.NewServer(handler)
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	requestJSON(t, client, http.MethodPost, server.URL+"/api/register", map[string]interface{}{
		"email": "member@example.com", "password": "member-password-42",
	}, http.StatusCreated)
	requestJSON(t, client, http.MethodPost, server.URL+"/api/login", map[string]interface{}{
		"email": "member@example.com", "password": "member-password-42",
	}, http.StatusOK)
	requestJSON(t, client, http.MethodPost, server.URL+"/api/shorten", map[string]interface{}{
		"original_url": "https://example.com/article", "custom_alias": "e2e-link",
	}, http.StatusCreated)
	requestJSON(t, client, http.MethodPost, server.URL+"/api/shorten", map[string]interface{}{
		"original_url": "https://example.com/article", "custom_alias": "another-alias",
	}, http.StatusOK)
	var destinationCount int
	if err := db.GetContext(ctx, &destinationCount, `
		SELECT COUNT(*)
		FROM url_info
		WHERE original_url = 'https://example.com/article' AND user_id IS NOT NULL
	`); err != nil || destinationCount != 1 {
		t.Fatalf("destination link count = %d, err = %v", destinationCount, err)
	}
	redirectResponse, err := client.Get(server.URL + "/e2e-link")
	if err != nil {
		t.Fatalf("request redirect: %v", err)
	}
	redirectResponse.Body.Close()
	if redirectResponse.StatusCode != http.StatusFound || redirectResponse.Header.Get("Location") != "https://example.com/article" {
		t.Fatalf("redirect status = %d, location = %q", redirectResponse.StatusCode, redirectResponse.Header.Get("Location"))
	}
	requestJSON(t, client, http.MethodPost, server.URL+"/api/user/change-password", map[string]interface{}{
		"old_password": "member-password-42", "new_password": "new-member-password-43",
	}, http.StatusOK)
	requestJSON(t, client, http.MethodGet, server.URL+"/api/me", nil, http.StatusUnauthorized)
	requestJSON(t, client, http.MethodPost, server.URL+"/api/login", map[string]interface{}{
		"email": "member@example.com", "password": "new-member-password-43",
	}, http.StatusOK)
	requestJSON(t, client, http.MethodDelete, server.URL+"/api/user/account", nil, http.StatusOK)
	deletedResponse, err := client.Get(server.URL + "/e2e-link")
	if err != nil {
		t.Fatalf("request deleted link: %v", err)
	}
	deletedResponse.Body.Close()
	if deletedResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted link status = %d", deletedResponse.StatusCode)
	}
}

func requestJSON(t *testing.T, client *http.Client, method, requestURL string, body map[string]interface{}, expectedStatus int) {
	t.Helper()
	var requestBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	request, err := http.NewRequest(method, requestURL, &requestBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("perform %s %s: %v", method, requestURL, err)
	}
	defer response.Body.Close()
	if response.StatusCode != expectedStatus {
		var payload map[string]interface{}
		_ = json.NewDecoder(response.Body).Decode(&payload)
		t.Fatalf("%s %s status = %d, payload = %v", method, requestURL, response.StatusCode, payload)
	}
}
