package account

import (
	"encoding/base64"
	"testing"
)

func jwtToken(payload string) string {
	return "hdr." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".sig"
}

func TestApplyIdentity(t *testing.T) {
	tok := jwtToken(`{"sub":"u1","nickname":"Hosea","preferred_username":"17673040926","email":"a@b.c"}`)
	id := applyIdentity(tok)
	if id.Nickname != "Hosea" || id.PreferredUsername != "17673040926" || id.Email != "a@b.c" {
		t.Errorf("identity wrong: %+v", id)
	}
}

func TestApplyIdentityFallsBackToPreferredUsername(t *testing.T) {
	tok := jwtToken(`{"sub":"u1","preferred_username":"17673040926"}`)
	id := applyIdentity(tok)
	if id.Nickname != "17673040926" {
		t.Errorf("nickname should fall back to preferred_username, got %q", id.Nickname)
	}
}

func TestApplyIdentityNonJWT(t *testing.T) {
	if id := applyIdentity("not-a-jwt"); id != (Identity{}) {
		t.Errorf("expected zero identity, got %+v", id)
	}
}

func TestOAuthUserID(t *testing.T) {
	if got := oauthUserID("tok", "acc123"); got != "uid_acc123" {
		t.Errorf("with account uid: %q", got)
	}
	got := oauthUserID("tok", "")
	if len(got) != len("oauth_")+12 || got[:6] != "oauth_" {
		t.Errorf("fallback hash id wrong: %q", got)
	}
	if oauthUserID("tok", "") != oauthUserID("tok", "") {
		t.Error("hash fallback must be deterministic")
	}
	if oauthUserID("tok2", "") == oauthUserID("tok", "") {
		t.Error("different tokens must hash differently")
	}
}

func TestExtractUserIDFromJWT(t *testing.T) {
	if got, ok := extractUserIDFromJWT(jwtToken(`{"sub":"abc"}`)); !ok || got != "abc" {
		t.Errorf("extract sub: %q %v", got, ok)
	}
	if _, ok := extractUserIDFromJWT("nope"); ok {
		t.Error("expected failure on non-jwt")
	}
}

func TestExtractIssuerInfo(t *testing.T) {
	tok := jwtToken(`{"iss":"https://copilot.tencent.com/sso-ent42"}`)
	domain, ent, ok := extractIssuerInfo(tok)
	if !ok || domain != "copilot.tencent.com" || ent != "ent42" {
		t.Errorf("issuer info: domain=%q ent=%q ok=%v", domain, ent, ok)
	}
}
