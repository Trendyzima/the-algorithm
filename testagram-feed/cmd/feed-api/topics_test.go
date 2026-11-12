package main

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func sendTopicRequest(handler http.Handler, body string, token bool) *httptest.ResponseRecorder {
    req := httptest.NewRequest(http.MethodPost, "/v1/topics/rank", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Testagram-API-Version", "1")
    req.Header.Set("X-Request-ID", "topic-request-1")
    if token { req.Header.Set("Authorization", "Bearer "+testToken) }
    response := httptest.NewRecorder()
    handler.ServeHTTP(response, req)
    return response
}

const validTopicRequest = `{"limit":2,"now":"2026-11-12T10:00:00Z","candidates":[{"tag":"testagram","postsCount":30,"lastUsedAt":"2026-11-12T09:00:00Z","interestAffinity":0.8},{"tag":"kenya","postsCount":100,"lastUsedAt":"2026-11-12T08:00:00Z","interestAffinity":0.2}]}`

func TestTopicRankRequiresInternalBearerToken(t *testing.T) {
    if got := sendTopicRequest(newHandler(testToken), validTopicRequest, false).Code; got != http.StatusUnauthorized { t.Fatalf("expected 401, got %d", got) }
}

func TestTopicRankReturnsRankedTopicsAndSecurityHeaders(t *testing.T) {
    response := sendTopicRequest(newHandler(testToken), validTopicRequest, true)
    if response.Code != http.StatusOK { t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String()) }
    if !strings.Contains(response.Body.String(), `"tag":"testagram"`) { t.Fatalf("personalized topic missing: %s", response.Body.String()) }
    if response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("X-Content-Type-Options") != "nosniff" { t.Fatalf("security headers missing: %#v", response.Header()) }
}

func TestTopicRankRejectsInvalidCandidates(t *testing.T) {
    cases := map[string]string{
        "invalid tag": `{"limit":1,"candidates":[{"tag":"bad tag","postsCount":10,"lastUsedAt":"2026-11-12T09:00:00Z","interestAffinity":0.5}]}` ,
        "zero volume": `{"limit":1,"candidates":[{"tag":"tag","postsCount":0,"lastUsedAt":"2026-11-12T09:00:00Z","interestAffinity":0.5}]}` ,
        "affinity outside range": `{"limit":1,"candidates":[{"tag":"tag","postsCount":10,"lastUsedAt":"2026-11-12T09:00:00Z","interestAffinity":1.1}]}` ,
        "duplicate tags": `{"limit":2,"candidates":[{"tag":"Tag","postsCount":10,"lastUsedAt":"2026-11-12T09:00:00Z","interestAffinity":0.5},{"tag":"tag","postsCount":20,"lastUsedAt":"2026-11-12T09:00:00Z","interestAffinity":0.5}]}` ,
    }
    for name, body := range cases {
        t.Run(name, func(t *testing.T) {
            response := sendTopicRequest(newHandler(testToken), body, true)
            if response.Code != http.StatusBadRequest { t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String()) }
        })
    }
}

func TestTopicRankRejectsTrailingJSON(t *testing.T) {
    response := sendTopicRequest(newHandler(testToken), validTopicRequest+" {}", true)
    if response.Code != http.StatusBadRequest { t.Fatalf("expected 400, got %d", response.Code) }
}
