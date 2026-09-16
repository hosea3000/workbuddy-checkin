package codebuddy

import (
	"context"
	"net/http"
	"testing"
)

func TestFetchModelsSuccess(t *testing.T) {
	c := quotaClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/config" {
			t.Errorf("path=%s", r.URL.Path)
		}
		if r.Header.Get("X-IDE-Type") != "CodeBuddyIDE" {
			t.Errorf("应使用 IDE 变体头集，X-IDE-Type=%q", r.Header.Get("X-IDE-Type"))
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"models":["glm-5.2",{"id":"deepseek-v4-pro"},"glm-5.2"]}}`))
	})
	models, err := c.FetchModels(context.Background(), testSnap())
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if len(models) != 2 || models[0] != "glm-5.2" || models[1] != "deepseek-v4-pro" {
		t.Fatalf("模型列表不符: %#v", models)
	}
}

func TestFetchModelsRejectsNonZeroCode(t *testing.T) {
	c := quotaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":10001,"data":{"models":["x"]}}`))
	})
	if _, err := c.FetchModels(context.Background(), testSnap()); err == nil {
		t.Fatal("非 0 业务码应返回错误")
	}
}

func TestFetchModelsMapsCredInvalid(t *testing.T) {
	c := quotaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	})
	_, err := c.FetchModels(context.Background(), testSnap())
	ue, ok := err.(*UpstreamError)
	if !ok {
		t.Fatalf("期望 UpstreamError，得到 %T", err)
	}
	if !ue.CredInvalid {
		t.Fatalf("401 应标记 CredInvalid: %#v", ue)
	}
}

func TestExtractModelIDs(t *testing.T) {
	got := ExtractModelIDs([]any{
		"a",
		map[string]any{"id": "b"},
		map[string]any{"model": "c"},
		map[string]any{"other": "d"},
		"",
		"a",
		nil,
	})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("得到 %#v，期望 %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("得到 %#v，期望 %#v", got, want)
		}
	}
}
