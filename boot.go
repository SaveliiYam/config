package config

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Boot — бутстрап из .env/env: какое окружение, сервис и какие секции включать.
// Настоящие настройки лежат в config/<env>/<service>/*.json (см. LoadSection).
type Boot struct {
	Env         string   `env:"APP_ENV" envDefault:"dev"`
	ServiceName string   `env:"SERVICE_NAME"`
	Dir         string   `env:"CONFIG_DIR" envDefault:"config"`
	Files       []string `env:"CONFIG_FILES"`   // белый список без .json; пусто = все
	Exclude     []string `env:"CONFIG_EXCLUDE"` // чёрный список поверх белого
}

type bootOptions struct {
	defaultService string
}

// BootOption настраивает LoadBoot.
type BootOption func(*bootOptions)

// WithService задаёт fallback для SERVICE_NAME, если переменная не задана.
func WithService(name string) BootOption {
	return func(o *bootOptions) { o.defaultService = name }
}

// LoadBoot читает .env (если есть) и поверх — переменные окружения.
// Путь к .env: ENV_FILE или ".env". Отсутствие файла — не ошибка.
func LoadBoot(opts ...BootOption) (*Boot, error) {
	o := bootOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	if err := loadDotEnv(envFile); err != nil {
		return nil, err
	}

	boot, err := Load[Boot]()
	if err != nil {
		return nil, err
	}
	if boot.ServiceName == "" {
		boot.ServiceName = o.defaultService
	}
	if boot.Env == "" || boot.ServiceName == "" {
		return nil, fmt.Errorf("config: APP_ENV and SERVICE_NAME must not be empty")
	}
	return boot, nil
}

// ConfigDir — папка конфигов: <Dir>/<Env>/<ServiceName>.
func (b *Boot) ConfigDir() string {
	return filepath.Join(b.Dir, b.Env, b.ServiceName)
}

// Enabled — включена ли секция name (без расширения) белым/чёрным списком.
func (b *Boot) Enabled(name string) bool {
	for _, ex := range b.Exclude {
		if strings.EqualFold(strings.TrimSpace(ex), name) {
			return false
		}
	}
	if len(b.Files) == 0 {
		return true
	}
	for _, f := range b.Files {
		if strings.EqualFold(strings.TrimSpace(f), name) {
			return true
		}
	}
	return false
}

// UnknownSections — имена из CONFIG_FILES, которых нет в known.
// Bootstrap должен упасть на них: опечатка в белом списке не должна
// молча превращаться в «секция выключена».
func (b *Boot) UnknownSections(known ...string) []string {
	set := make(map[string]bool, len(known))
	for _, k := range known {
		set[strings.ToLower(k)] = true
	}
	var unknown []string
	for _, f := range b.Files {
		f = strings.TrimSpace(f)
		if f != "" && !set[strings.ToLower(f)] {
			unknown = append(unknown, f)
		}
	}
	return unknown
}

// LoadSection читает <ConfigDir>/<name>.json в out.
//
// Секция выключена — (false, nil), out не трогает (дефолты слоя).
// Включена, но файла нет — ошибка. Неизвестный ключ JSON — ошибка.
func LoadSection[T any](b *Boot, name string, out *T) (bool, error) {
	if !b.Enabled(name) {
		return false, nil
	}
	path := filepath.Join(b.ConfigDir(), name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("config: section %q enabled but not readable (%s): %w", name, path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return false, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return true, nil
}

// AppConfig — типичная секция app.json.
type AppConfig struct {
	Name     string `json:"name"`
	LogLevel string `json:"log_level"`
}

// Configurable — слой сам применяет свой конфиг через Configure.
// Слою не нужно импортировать этот пакет: соответствие структурное в Apply.
type Configurable[T any] interface {
	Configure(ctx context.Context, cfg T) error
}

// Apply отдаёт конфиг слою, если тот реализует Configurable[T]; иначе nil.
func Apply[T any](ctx context.Context, target any, cfg T) error {
	if c, ok := target.(Configurable[T]); ok {
		return c.Configure(ctx, cfg)
	}
	return nil
}

// loadDotEnv подставляет значения из файла в окружение, не перетирая уже заданные.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, alreadySet := os.LookupEnv(key); !alreadySet {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("config: %s: %w", path, err)
			}
		}
	}
	return sc.Err()
}
