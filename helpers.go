package config

import (
	"context"
	"fmt"
)

// Open — LoadBoot(service) + секция app.json.
func Open(service string, opts ...BootOption) (*Boot, AppConfig, error) {
	boot, err := LoadBoot(append([]BootOption{WithService(service)}, opts...)...)
	if err != nil {
		return nil, AppConfig{}, err
	}
	appCfg := AppConfig{LogLevel: "info"}
	if _, err := LoadSection(boot, "app", &appCfg); err != nil {
		return nil, AppConfig{}, err
	}
	return boot, appCfg, nil
}

// Section читает секцию поверх def (выключена — вернёт def без ошибки).
func Section[T any](b *Boot, name string, def T) (T, error) {
	out := def
	if _, err := LoadSection(b, name, &out); err != nil {
		return def, err
	}
	return out, nil
}

// Require падает, если в CONFIG_FILES есть имена вне known.
func Require(b *Boot, known ...string) error {
	if unknown := b.UnknownSections(known...); len(unknown) > 0 {
		return fmt.Errorf("config: unknown CONFIG_FILES sections: %v", unknown)
	}
	return nil
}

// LoadApply — Section + Apply к Configurable-слою.
func LoadApply[T any](ctx context.Context, b *Boot, name string, target any, def T) error {
	cfg, err := Section(b, name, def)
	if err != nil {
		return err
	}
	return Apply(ctx, target, cfg)
}
