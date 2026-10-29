# Proba — backend

Backend платформы Proba: приложение для осознанного потребления напитков (кофе, пиво, вино). Ядро — модель «проба» (дегустация: оценка, оси вкуса, колесо вкусов, место, компания); поверх неё вкусовой профиль, паспорт, лента, заведения, рекомендации.

## Стек

Go, zerolog · Yandex Serverless Containers · YDB Serverless · Object Storage · API Gateway · Yandex Message Queue · Yandex Monitoring + Grafana Cloud. Мобильный клиент: React Native + Expo (TypeScript) — отдельный репозиторий.

## Источник правды

- [`agent-instructions/AGENT_BRIEF.md`](agent-instructions/AGENT_BRIEF.md) — бриф: продукт, архитектура, фазы, конвенции.
- [`agent-instructions/openapi.yaml`](agent-instructions/openapi.yaml) — контракт (OpenAPI 3.0.3): операции с `x-phase`, `x-flag`, `x-module`, реестр флагов `x-flags`.

Контракт и код меняются одним коммитом; фича-флаги включают/выключают операции, клиент прячет выключенное.

## Структура

```
cmd/api/                    composition root: собирает модули, поднимает HTTP
internal/platform/          config, zerolog, ydb-клиент, http, middleware, flags, events, idempotency
internal/modules/<name>/    api.go — публичный интерфейс модуля + DTO; module.go — New(deps) -> Module
    internal/               domain, app, store, transport/http
migrations/<name>/          миграции по модулям
spikes/                     заметки по спайкам (YDB)
```

## CI

Проверки: `go vet`, `golangci-lint` (включая `depguard` — изоляция модулей), `go test ./...`. Конфигурация — `.golangci.yml`, пайплайн — `.sourcecraft/ci.yaml`.
