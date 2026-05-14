package handlers

import (
	"net/http"
	"testing"

	"github.com/elazarl/goproxy"
	"github.com/stretchr/testify/assert"
)

func newNoopHandler() *ExternalNoopHandler {
	return NewExternalNoopHandler("internal.mirror.corp", "example.corp")
}

// --- Actions re-routing ---

func TestActionsCheckoutIsReRouted(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://github.com/actions/checkout", nil)
	outReq, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.Nil(t, resp, "actions request should not get a synthetic response")
	assert.Equal(t, "internal.mirror.corp", outReq.URL.Host)
	assert.Equal(t, "internal.mirror.corp", outReq.Host)
}

func TestActionsAPIRequestIsReRouted(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/actions/checkout/releases", nil)
	outReq, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.Nil(t, resp)
	assert.Equal(t, "internal.mirror.corp", outReq.URL.Host)
}

func TestActionsCodeloadIsReRouted(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://codeload.github.com/actions/checkout/tar.gz/refs/heads/main", nil)
	outReq, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.Nil(t, resp)
	assert.Equal(t, "internal.mirror.corp", outReq.URL.Host)
}

// --- Allowed domain passthrough ---

func TestAllowedDomainPassesThrough(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.corp/some/path", nil)
	_, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.Nil(t, resp, "allowed domain should pass through")
}

func TestAllowedSubdomainPassesThrough(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://registry.example.corp/packages", nil)
	_, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.Nil(t, resp, "subdomain of allowed domain should pass through")
}

func TestDeepSubdomainPassesThrough(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://a.b.example.corp/packages", nil)
	_, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.Nil(t, resp, "deep subdomain should pass through")
}

// --- Blocked cases ---

func TestFakeSuffixIsBlocked(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://evilexample.corp/packages", nil)
	_, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestExternalHostIsBlocked(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://registry.npmjs.org/lodash", nil)
	_, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNonActionsGitHubIsBlocked(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://github.com/some-org/some-repo", nil)
	_, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMyActionsOrgIsBlocked(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://github.com/my-actions/tool", nil)
	_, resp := newNoopHandler().HandleRequest(req, &goproxy.ProxyCtx{})
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestEmptyAllowedDomainBlocksEverything(t *testing.T) {
	h := NewExternalNoopHandler("internal.mirror.corp", "")
	req, _ := http.NewRequest("GET", "https://anything.example.com/path", nil)
	_, resp := h.HandleRequest(req, &goproxy.ProxyCtx{})
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
