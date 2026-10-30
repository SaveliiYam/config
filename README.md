# config — env + JSON-секции сервиса

Два уровня без внешних зависимостей:

1. **Env** — `Load[T]()` по тегам `env` / `envDefault` / `envRequired` / `envPrefix`.
2. **Boot + секции** — `.env` выбирает окружение/сервис/файлы; настройки слоёв в
   `config/<env>/<service>/*.json` через `LoadSection`.

```go
boot, appCfg, err := config.Open("jwt")
jwtCfg, err := config.Section(boot, "jwt", service.DefaultConfig())
_ = config.LoadApply(ctx, boot, "http", transport, ginserver.DefaultModuleConfig())
_ = config.Require(boot, "app", "http", "jwt", "basic_auth")
```

Пустой `CONFIG_FILES` — читать всё; имя в списке, но файла нет — ошибка при старте;
`Require` ловит опечатки в белом списке.

## Установка

```bash
go get github.com/SaveliiYam/config@v0.2.0
```

Требуется Go ≥ 1.25.
