# config — конфиг из env

Загрузка структуры из переменных окружения по тегам `env`, `envDefault`, `envRequired`, `envPrefix`. Stdlib only.

```go
cfg, err := config.Load[Config]()
```

## Установка

```bash
go get github.com/SaveliiYam/config@v0.1.0
```

Требуется Go ≥ 1.25. Лицензия: см. репозиторий.
