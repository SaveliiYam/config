package config_test

import (
	"testing"
	"time"

	"github.com/SaveliiYam/config"
)

type cfg struct {
	HTTP struct {
		Addr string `env:"ADDR" envDefault:":8080"`
	} `envPrefix:"HTTP_"`
	DSN     string        `env:"DATABASE_URL" envRequired:"true"`
	Timeout time.Duration `env:"TIMEOUT" envDefault:"5s"`
	Tags    []string      `env:"TAGS" envDefault:"a,b"`
}

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("HTTP_ADDR", ":9000")
	got, err := config.Load[cfg]()
	if err != nil {
		t.Fatal(err)
	}
	if got.HTTP.Addr != ":9000" || got.Timeout != 5*time.Second || len(got.Tags) != 2 {
		t.Fatalf("%+v", got)
	}
	if _, err := config.Load[struct {
		X string `env:"DEFINITELY_MISSING" envRequired:"true"`
	}](); err == nil {
		t.Fatal("expected required error")
	}
}
