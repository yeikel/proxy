package handlers

import (
	"net/http"
	"strings"

	"github.com/dependabot/proxy/internal/helpers"
	"github.com/elazarl/goproxy"
	"github.com/sirupsen/logrus"
)

// ExternalNoopHandler blocks all external requests with an empty 200 response.
// Two exceptions are forwarded normally:
//  1. github.com/actions/* requests are rewritten to an internal mirror host.
//  2. Requests to the configured allowed domain (and any subdomain) pass through.
type ExternalNoopHandler struct {
	actionsInternalHost string
	allowedDomain       string
}

func NewExternalNoopHandler(actionsInternalHost, allowedDomain string) *ExternalNoopHandler {
	return &ExternalNoopHandler{
		actionsInternalHost: actionsInternalHost,
		allowedDomain:       allowedDomain,
	}
}

var githubActionsHosts = []string{
	"github.com",
	"api.github.com",
	"codeload.github.com",
	"raw.githubusercontent.com",
}

func hostInList(host string, list []string) bool {
	for _, h := range list {
		if strings.EqualFold(host, h) {
			return true
		}
	}
	return false
}

// isActionsRequest returns true when the request targets the "actions" GitHub org.
// Handles both github.com/{owner}/... and api.github.com/repos/{owner}/... path shapes.
func isActionsRequest(req *http.Request) bool {
	host := helpers.GetHost(req)
	if !hostInList(host, githubActionsHosts) {
		return false
	}

	parts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/"), "/", 3)
	if len(parts) == 0 {
		return false
	}

	owner := parts[0]
	if host == "api.github.com" && owner == "repos" && len(parts) >= 2 {
		owner = parts[1]
	}

	return strings.EqualFold(owner, "actions")
}

// isAllowedDomain returns true if host exactly matches the allowed domain
// or is a proper subdomain of it (e.g. "registry.example.corp" for "example.corp").
// A host that merely shares a suffix (e.g. "evilexample.corp") is rejected.
func (h *ExternalNoopHandler) isAllowedDomain(host string) bool {
	if h.allowedDomain == "" {
		return false
	}
	return host == h.allowedDomain || strings.HasSuffix(host, "."+h.allowedDomain)
}

func (h *ExternalNoopHandler) HandleRequest(
	req *http.Request,
	ctx *goproxy.ProxyCtx,
) (*http.Request, *http.Response) {
	host := helpers.GetHost(req)

	// 1. Actions requests: rewrite host to internal mirror and forward.
	if isActionsRequest(req) {
		logrus.Debugf("* re-routing actions request %s to %s", req.URL.Path, h.actionsInternalHost)
		req.URL.Host = h.actionsInternalHost
		req.Host = h.actionsInternalHost
		return req, nil
	}

	// 2. Allowed domain and subdomains: forward normally.
	if h.isAllowedDomain(host) {
		logrus.Debugf("* allowing request to %s", host)
		return req, nil
	}

	// 3. Everything else: empty 200, no connection made.
	logrus.Debugf("* blocking external request to %s", host)
	return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusOK, "")
}
