package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
)

const (
	credentialsFile = "credentials.json"
	settingsFile    = "settings.json"
)

// Store 是 JSON 文件持久化层：原子写 + 损坏自愈 + 内存缓存。
// 单进程单实例，锁只防 UI 与调度器并发。
type Store struct {
	dir string

	mu          sync.Mutex
	credentials []model.Credential
	settings    model.Settings
}

// New 创建 Store。dir 为空时使用 os.UserConfigDir()/workbuddy-checkin。
func New(dir string) (*Store, error) {
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("resolve config dir: %w", err)
		}
		dir = filepath.Join(base, "workbuddy-checkin")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	s := &Store{dir: dir, settings: model.DefaultSettings()}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Dir 返回数据目录。
func (s *Store) Dir() string { return s.dir }

func (s *Store) load() error {
	if err := readJSON(filepath.Join(s.dir, credentialsFile), &s.credentials); err != nil {
		return err
	}
	// settings 缺失字段用默认值填充：先放默认，再覆盖文件里出现的字段。
	settings := model.DefaultSettings()
	if err := readJSON(filepath.Join(s.dir, settingsFile), &settings); err != nil {
		return err
	}
	s.settings = settings
	return nil
}

// readJSON 读取 JSON；文件不存在视为空（返回 nil）；解析失败则备份为 .corrupt-<ts> 并返回空。
func readJSON(path string, v any) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		backup := fmt.Sprintf("%s.corrupt-%d", path, time.Now().Unix())
		_ = os.Rename(path, backup)
		return nil
	}
	return nil
}

// writeJSON 原子写：临时文件 → fsync → rename，权限 0600。
func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// ListCredentials 返回凭证的深拷贝副本。
func (s *Store) ListCredentials() []model.Credential {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Credential, len(s.credentials))
	copy(out, s.credentials)
	return out
}

// GetCredential 按 ID 返回凭证副本。
func (s *Store) GetCredential(id string) (model.Credential, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.credentials {
		if c.ID == id {
			return c, true
		}
	}
	return model.Credential{}, false
}

// FindCredential 按谓词查找首个匹配凭证。
func (s *Store) FindCredential(match func(model.Credential) bool) (model.Credential, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.credentials {
		if match(c) {
			return c, true
		}
	}
	return model.Credential{}, false
}

// SaveCredential 插入或原位更新凭证，并立即落盘。
func (s *Store) SaveCredential(cred model.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	if cred.CreatedAt == 0 {
		cred.CreatedAt = now
	}
	cred.UpdatedAt = now
	replaced := false
	for i := range s.credentials {
		if s.credentials[i].ID == cred.ID {
			s.credentials[i] = cred
			replaced = true
			break
		}
	}
	if !replaced {
		s.credentials = append(s.credentials, cred)
	}
	return s.persistCredentialsLocked()
}

// DeleteCredential 删除凭证并落盘。
func (s *Store) DeleteCredential(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.credentials {
		if s.credentials[i].ID == id {
			s.credentials = append(s.credentials[:i], s.credentials[i+1:]...)
			return s.persistCredentialsLocked()
		}
	}
	return nil
}

func (s *Store) persistCredentialsLocked() error {
	return writeJSON(filepath.Join(s.dir, credentialsFile), s.credentials)
}

// GetSettings 返回设置副本。
func (s *Store) GetSettings() model.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

// SaveSettings 覆盖设置并落盘。
func (s *Store) SaveSettings(settings model.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
	return writeJSON(filepath.Join(s.dir, settingsFile), s.settings)
}
