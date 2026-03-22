package auth_test

import (
	"net/http/httptest"
	"os"
	"testing"
)

// GET /api/v1/auth/logout

func TestLogout_RedirectsToConfiguredURL(t *testing.T) {
	const logoutURL = "https://team.cloudflareaccess.com/cdn-cgi/access/logout"
	t.Setenv("LOGOUT_REDIRECT_URL", logoutURL)

	resp, err := newApp(nil).Test(httptest.NewRequest("GET", "/api/v1/auth/logout", nil))
	if err != nil {
		t.Fatal(err)
	}

	assertStatus(t, 302, resp)
	assertLocation(t, logoutURL, resp)
}

func TestLogout_LocalDevURLWorks(t *testing.T) {
	const logoutURL = "/cdn-cgi/access/logout"
	t.Setenv("LOGOUT_REDIRECT_URL", logoutURL)

	resp, err := newApp(nil).Test(httptest.NewRequest("GET", "/api/v1/auth/logout", nil))
	if err != nil {
		t.Fatal(err)
	}

	assertStatus(t, 302, resp)
	assertLocation(t, logoutURL, resp)
}

func TestLogout_Returns500WhenURLNotConfigured(t *testing.T) {
	os.Unsetenv("LOGOUT_REDIRECT_URL")

	resp, err := newApp(nil).Test(httptest.NewRequest("GET", "/api/v1/auth/logout", nil))
	if err != nil {
		t.Fatal(err)
	}

	assertStatus(t, 500, resp)
}
