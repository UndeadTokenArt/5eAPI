package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNavigationRenderedOnAppPages(t *testing.T) {
	tests := []struct {
		path string
		link string
	}{
		{"/neotokyo", "/characters"},
		{"/characters", "/campaigns"},
		{"/campaigns", "/characters"},
		{"/editor", "/neotokyo"},
		{"/map", "/map/editor"},
		{"/map/editor", "/map"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			request.Header.Set("Remote-User", "navigation-test")
			request.Header.Set("Remote-Groups", "editor")
			response := httptest.NewRecorder()
			setupRouter().ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			checkNavigationMarkup(t, response.Body.String(), tt.link)
		})
	}
}

func TestMainNavigationTemplate(t *testing.T) {
	var body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&body, "home", []string{"spells"}); err != nil {
		t.Fatal(err)
	}
	checkNavigationMarkup(t, body.String(), "/encounter")
}

func checkNavigationMarkup(t *testing.T, body, link string) {
	t.Helper()
	for _, fragment := range []string{
		`/static/css/navigation.css`,
		`/static/js/navigation.js`,
		`aria-label="Main navigation"`,
		`class="nav-toggle"`,
		`aria-controls="main-nav-links"`,
		`aria-expanded="false"`,
		`class="nav-links" id="main-nav-links"`,
		`href="` + link + `"`,
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("page is missing navigation markup %q", fragment)
		}
	}
}
