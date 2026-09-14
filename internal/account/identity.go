package account

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// Identity 是 JWT payload 中用于展示的用户身份字段。
type Identity struct {
	Nickname          string
	PreferredUsername string
	Email             string
}

// applyIdentity 从 JWT payload 提取昵称/邮箱等展示字段；payload 不可解析时返回零值。
// 回退规则：昵称缺失时用 preferred_username。
func applyIdentity(bearerToken string) Identity {
	payload, err := jwtPayload(bearerToken)
	if err != nil {
		return Identity{}
	}
	var id Identity
	if v, _ := payload["nickname"].(string); v != "" {
		id.Nickname = v
	}
	if v, _ := payload["preferred_username"].(string); v != "" {
		id.PreferredUsername = v
	}
	if v, _ := payload["email"].(string); v != "" {
		id.Email = v
	}
	if id.Nickname == "" && id.PreferredUsername != "" {
		id.Nickname = id.PreferredUsername
	}
	return id
}

// oauthUserID 生成 OAuth 凭证的稳定 user_id：优先 uid_<account_uid>，缺失时退回 oauth_<token 哈希>。
func oauthUserID(accessToken, accountUID string) string {
	if accountUID != "" {
		return "uid_" + accountUID
	}
	return "oauth_" + shortHash(accessToken)
}

// shortHash 取 token 的 SHA-256 前 12 位（hex）。
func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:12]
}

// extractUserIDFromJWT 解析 JWT payload 的 sub 字段。
func extractUserIDFromJWT(token string) (string, bool) {
	payload, err := jwtPayload(token)
	if err != nil {
		return "", false
	}
	if sub, ok := payload["sub"].(string); ok && sub != "" {
		return sub, true
	}
	return "", false
}

// extractIssuerInfo 从 JWT iss 提取 domain 与 sso-<enterprise_id>。
func extractIssuerInfo(token string) (domain, enterpriseID string, ok bool) {
	payload, err := jwtPayload(token)
	if err != nil {
		return "", "", false
	}
	iss, _ := payload["iss"].(string)
	if iss == "" {
		return "", "", false
	}
	parts := strings.SplitN(iss, "://", 2)
	rest := parts[len(parts)-1]
	host := rest
	if idx := strings.Index(rest, "/"); idx >= 0 {
		host = rest[:idx]
		rest = rest[idx:]
	} else {
		rest = ""
	}
	host = strings.Split(host, ":")[0]
	if p := strings.Trim(rest, "/"); p != "" {
		segs := strings.Split(p, "/")
		last := segs[len(segs)-1]
		if strings.HasPrefix(last, "sso-") && len(last) > len("sso-") {
			enterpriseID = last[len("sso-"):]
		}
	}
	return host, enterpriseID, true
}

// jwtPayload 解析 JWT 第二段（payload），容忍缺失的 padding。
func jwtPayload(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, errNotJWT
	}
	payloadPart := parts[1]
	if m := len(payloadPart) % 4; m != 0 {
		payloadPart += strings.Repeat("=", 4-m)
	}
	raw, err := base64.URLEncoding.DecodeString(payloadPart)
	if err != nil {
		raw, err = base64.RawURLEncoding.DecodeString(strings.TrimRight(payloadPart, "="))
		if err != nil {
			return nil, err
		}
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

var errNotJWT = &jwtError{"not a jwt"}

type jwtError struct{ msg string }

func (e *jwtError) Error() string { return e.msg }
