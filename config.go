// Package config — загрузка конфига из переменных окружения без зависимостей.
//
//	type Config struct {
//		HTTP struct {
//			Addr string `env:"ADDR" envDefault:":8080"`
//		} `envPrefix:"HTTP_"`
//		DatabaseURL string `env:"DATABASE_URL" envRequired:"true"`
//		Debug       bool   `env:"DEBUG" envDefault:"false"`
//	}
//
//	cfg, err := config.Load[Config]()
package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Load читает env в новую структуру T.
func Load[T any]() (*T, error) {
	var cfg T
	if err := LoadInto(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadInto заполняет структуру по тегам env/envDefault/envRequired/envPrefix/envSeparator.
func LoadInto(dst any) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config: dst must be pointer to struct")
	}
	return fill(v.Elem(), "")
}

func fill(v reflect.Value, prefix string) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f, fv := t.Field(i), v.Field(i)
		if !fv.CanSet() {
			continue
		}
		if fv.Kind() == reflect.Struct && f.Type != reflect.TypeOf(time.Time{}) {
			if err := fill(fv, prefix+f.Tag.Get("envPrefix")); err != nil {
				return err
			}
			continue
		}
		name := f.Tag.Get("env")
		if name == "" {
			continue
		}
		raw, ok := os.LookupEnv(prefix + name)
		if !ok {
			raw, ok = f.Tag.Lookup("envDefault")
		}
		if !ok || raw == "" {
			if f.Tag.Get("envRequired") == "true" {
				return fmt.Errorf("config: required env %s is not set", prefix+name)
			}
			continue
		}
		sep := f.Tag.Get("envSeparator")
		if sep == "" {
			sep = ","
		}
		if err := set(fv, raw, sep); err != nil {
			return fmt.Errorf("config: env %s: %w", prefix+name, err)
		}
	}
	return nil
}

func set(fv reflect.Value, raw, sep string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fv.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return err
			}
			fv.SetInt(int64(d))
			return nil
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(n)
	case reflect.Slice:
		if fv.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice type %s", fv.Type())
		}
		if raw == "" {
			fv.Set(reflect.ValueOf([]string{}))
			return nil
		}
		parts := strings.Split(raw, sep)
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		fv.Set(reflect.ValueOf(parts))
	default:
		return fmt.Errorf("unsupported type %s", fv.Type())
	}
	return nil
}
