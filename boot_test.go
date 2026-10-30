package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvDoesNotOverrideRealEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\nAPP_ENV=dev\nexport SERVICE_NAME=svc\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "prod")

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("APP_ENV"); got != "prod" {
		t.Fatalf("APP_ENV = %s, want prod", got)
	}
	if got := os.Getenv("SERVICE_NAME"); got != "svc" {
		t.Fatalf("SERVICE_NAME = %s, want svc", got)
	}
}

func TestLoadDotEnvMissingFileIsOK(t *testing.T) {
	if err := loadDotEnv(filepath.Join(t.TempDir(), "absent.env")); err != nil {
		t.Fatal(err)
	}
}

func TestLoadBootWithServiceFallback(t *testing.T) {
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "absent.env"))
	t.Setenv("APP_ENV", "dev")
	t.Setenv("SERVICE_NAME", "")
	_ = os.Unsetenv("SERVICE_NAME")

	boot, err := LoadBoot(WithService("jwt"))
	if err != nil {
		t.Fatal(err)
	}
	if boot.ServiceName != "jwt" {
		t.Fatalf("ServiceName = %q", boot.ServiceName)
	}
}

type section struct {
	Kind string `json:"kind"`
}

func writeSection(t *testing.T, boot *Boot, name, body string) {
	t.Helper()
	dir := boot.ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSectionWhitelist(t *testing.T) {
	boot := &Boot{Env: "dev", ServiceName: "svc", Dir: t.TempDir(), Files: []string{"repository"}}
	writeSection(t, boot, "repository", `{"kind":"postgres"}`)
	writeSection(t, boot, "database", `{"kind":"ignored"}`)

	out := section{Kind: "memory"}
	loaded, err := LoadSection(boot, "repository", &out)
	if err != nil || !loaded || out.Kind != "postgres" {
		t.Fatalf("repository: loaded=%v out=%+v err=%v", loaded, out, err)
	}
	out2 := section{Kind: "memory"}
	loaded, err = LoadSection(boot, "database", &out2)
	if err != nil || loaded || out2.Kind != "memory" {
		t.Fatalf("database: loaded=%v out=%+v err=%v", loaded, out2, err)
	}
	boot.Files = append(boot.Files, "missing")
	if _, err := LoadSection(boot, "missing", &out); err == nil {
		t.Fatal("expected error for enabled-but-missing section")
	}
}

func TestLoadSectionRejectsUnknownKeyAndBadJSON(t *testing.T) {
	boot := &Boot{Env: "dev", ServiceName: "svc", Dir: t.TempDir()}
	out := section{}
	writeSection(t, boot, "repository", `{"kind":"memory","knd":"typo"}`)
	if _, err := LoadSection(boot, "repository", &out); err == nil {
		t.Fatal("expected error for unknown key")
	}
	writeSection(t, boot, "repository", `{"kind":`)
	if _, err := LoadSection(boot, "repository", &out); err == nil {
		t.Fatal("expected error for malformed json")
	}
}

type fakeLayer struct{ got section }

func (f *fakeLayer) Configure(_ context.Context, cfg section) error { f.got = cfg; return nil }

func TestApplyOnlyWhenConfigurable(t *testing.T) {
	layer := &fakeLayer{}
	if err := Apply(context.Background(), layer, section{Kind: "postgres"}); err != nil {
		t.Fatal(err)
	}
	if layer.got.Kind != "postgres" {
		t.Fatalf("Configure not called: %+v", layer.got)
	}
	if err := Apply(context.Background(), struct{}{}, section{}); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownSections(t *testing.T) {
	boot := &Boot{Files: []string{"app", "typo"}}
	got := boot.UnknownSections("app", "http")
	if len(got) != 1 || got[0] != "typo" {
		t.Fatalf("%v", got)
	}
	if err := Require(boot, "app", "http"); err == nil {
		t.Fatal("expected Require error")
	}
}

func TestSectionDefaults(t *testing.T) {
	boot := &Boot{Env: "dev", ServiceName: "svc", Dir: t.TempDir(), Files: []string{"other"}}
	got, err := Section(boot, "jwt", AppConfig{LogLevel: "info"})
	if err != nil || got.LogLevel != "info" {
		t.Fatalf("%+v %v", got, err)
	}
}
