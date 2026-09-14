package codebuddy

import (
	"strings"
	"testing"
)

func TestGenerateHeadersStandard(t *testing.T) {
	cred := CredentialSnapshot{BearerToken: "tok", UserID: "u1"}
	headers, err := GenerateHeaders(cred, ConversationIDs{}, "2.107.0")
	if err != nil {
		t.Fatal(err)
	}
	required := []string{
		"Authorization", "X-User-Id", "X-Domain", "X-Product", "X-CodeBuddy-Request",
		"X-Agent-Intent", "X-Agent-Purpose", "X-IDE-Type", "X-IDE-Name", "X-IDE-Version",
		"x-stainless-lang", "x-stainless-runtime", "x-stainless-runtime-version",
		"X-Conversation-ID", "X-Conversation-Request-ID", "X-Conversation-Message-ID", "X-Request-ID",
		"User-Agent",
	}
	for _, k := range required {
		if headers[k] == "" {
			t.Errorf("missing header %s", k)
		}
	}
	if headers["X-Product"] != "SaaS" || headers["X-Agent-Intent"] != "craft" ||
		headers["X-IDE-Type"] != "CLI" || headers["x-stainless-lang"] != "js" ||
		headers["x-stainless-runtime"] != "node" {
		t.Errorf("constant header mismatch: %v", headers)
	}
	if headers["User-Agent"] != "CLI/2.107.0 CodeBuddy/2.107.0" {
		t.Errorf("user-agent wrong: %q", headers["User-Agent"])
	}
	if len(headers["X-Conversation-Request-ID"]) != 32 {
		t.Errorf("conversation request id must be hex16 (32 chars), got %d", len(headers["X-Conversation-Request-ID"]))
	}
	if strings.Contains(headers["X-Conversation-ID"], "-") {
		// uuid 格式正确
	} else if len(headers["X-Conversation-ID"]) != 32 {
		t.Errorf("conversation id must be uuid-like")
	}
}

func TestGenerateHeadersEnterprise(t *testing.T) {
	cred := CredentialSnapshot{BearerToken: "tok", UserID: "u1", EnterpriseID: "e1", DepartmentFullName: "a/b c"}
	headers, err := GenerateHeaders(cred, ConversationIDs{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if headers["X-Enterprise-Id"] != "e1" || headers["X-Tenant-Id"] != "e1" {
		t.Errorf("enterprise headers missing: %v", headers)
	}
	if headers["X-Department-Info"] != "a%2Fb%20c" {
		t.Errorf("department info not urlencoded: %q", headers["X-Department-Info"])
	}
}

func TestGenerateHeadersAccountUIDPreferred(t *testing.T) {
	headers, _ := GenerateHeaders(CredentialSnapshot{BearerToken: "t", UserID: "u1", AccountUID: "acc"}, ConversationIDs{}, "")
	if headers["X-User-Id"] != "acc" {
		t.Errorf("account_uid should take precedence, got %s", headers["X-User-Id"])
	}
}

func TestGenerateHeadersMissing(t *testing.T) {
	if _, err := GenerateHeaders(CredentialSnapshot{BearerToken: "t"}, ConversationIDs{}, ""); err == nil {
		t.Error("expected error when user_id missing")
	}
	if _, err := GenerateHeaders(CredentialSnapshot{UserID: "u"}, ConversationIDs{}, ""); err == nil {
		t.Error("expected error when bearer_token missing")
	}
}

func TestMapUpstreamStatus(t *testing.T) {
	if e := MapUpstreamStatus(401, "x"); !e.CredInvalid || e.StatusCode != 401 {
		t.Errorf("401 mapping wrong: %+v", e)
	}
	if e := MapUpstreamStatus(403, "x"); !e.CredInvalid {
		t.Errorf("403 should mark credential invalid")
	}
	if e := MapUpstreamStatus(429, "x"); e.StatusCode != 429 || e.ErrType != ErrCategoryRateLimit {
		t.Errorf("429 mapping wrong: %+v", e)
	}
	if e := MapUpstreamStatus(502, "x"); e.StatusCode != 502 || e.ErrType != ErrCategoryUpstream5xx {
		t.Errorf("5xx mapping wrong: %+v", e)
	}
	if e := MapUpstreamStatus(200, "x"); e.StatusCode != 502 || e.ErrType != ErrCategoryProtocol {
		t.Errorf("unexpected status mapping wrong: %+v", e)
	}
}

func TestParseUpstreamErrorBodyBusiness(t *testing.T) {
	msg, _, code := ParseUpstreamErrorBody(`{"code":12005,"msg":"err"}`)
	if code != 12005 || msg != "企业许可证没有可用席位" {
		t.Errorf("business mapping wrong: %q %v", msg, code)
	}
	msg2, _, _ := ParseUpstreamErrorBody(`{"error":{"message":"bad","type":"invalid_request_error"}}`)
	if msg2 != "bad" {
		t.Errorf("openai-style error parse wrong: %q", msg2)
	}
	msg3, _, _ := ParseUpstreamErrorBody("plain text")
	if msg3 != "plain text" {
		t.Errorf("fallback wrong: %q", msg3)
	}
}
