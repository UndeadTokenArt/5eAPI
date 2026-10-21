package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

// AccountInfo is what the navigation bar needs to render sign in / sign out.
type AccountInfo struct {
	SignedIn       bool   `json:"signed_in"`
	Guest          bool   `json:"guest"`
	Editor         bool   `json:"editor,omitempty"`
	User           string `json:"user,omitempty"`
	CharacterCount int    `json:"character_count"`
	LoginURL       string `json:"login_url,omitempty"`
	LogoutURL      string `json:"logout_url,omitempty"`
}

// autheliaURL returns the trusted portal base URL, or "" when unconfigured.
func autheliaURL() string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("AUTHELIA_URL")), "/")
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return ""
	}
	return base
}

func portalLink(base, path, returnTo string) string {
	if base == "" {
		return ""
	}
	link := base + path
	if returnTo != "" {
		link += "?rd=" + url.QueryEscape(returnTo)
	}
	return link
}

// returnURL rebuilds the public URL of the page that asked, from the proxy headers.
func returnURL(r *http.Request) string {
	page, err := url.Parse(r.Header.Get("Referer"))
	if err != nil || page.Host == "" || !strings.EqualFold(page.Host, r.Host) {
		return ""
	}
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme != "http" && scheme != "https" {
		scheme = page.Scheme
	}
	return scheme + "://" + page.Host + page.RequestURI()
}

func accountHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	user, guest := characterUser(w, r)
	info := AccountInfo{}
	base := autheliaURL()
	returnTo := returnURL(r)
	_, info.Editor = editorUser(r)
	if user != "" {
		info.SignedIn = !guest
		info.Guest = guest
		if !guest {
			info.User = user
			info.LogoutURL = portalLink(base, "/logout", returnTo)
		}
		if characters, err := savedCharacters.list(user); err == nil {
			info.CharacterCount = len(characters)
		}
	}
	if !info.SignedIn {
		info.LoginURL = portalLink(base, "/", returnTo)
	}
	writeCharacterJSON(w, http.StatusOK, info)
}
