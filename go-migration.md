# Go Migration: myinvesthelper_backend external requests

## Источник: FastAPI бэкенд
Path: `/Users/i.chutchev/Pycharm Projects/personal/myinvesthelper_backend`

## Статус на 2026-06-30

Обозначения: `[x]` — реализовано и подключено, `[~]` — реализовано частично, `[ ]` — не реализовано.

- [x] Каркас Go-сервиса на Fiber: конфигурация, HTTP-роуты, healthcheck, Swagger, middleware и graceful shutdown.
- [x] MOEX: HTTP-клиент для всех четырёх ISS endpoint, нормализация ответов и сервисный слой.
- [x] MOEX: конкурентный fan-out с ограничением до 8 запросов.
- [x] Кэш MOEX: Redis с TTL 15 минут по умолчанию для отдельных облигаций и market universe.
- [x] MOEX-сервис подключён к `GET /v1/bonds/{isin}` и `GET /v1/bonds`.
- [~] CBR: определены HTTP-контракт, DTO, сервисный интерфейс и `GET /v1/cbr/rates`, но клиент и сервис пока возвращают `ErrNotImplemented`.
- [ ] CBR: скачивание и парсинг HTML истории ставки, поиск `.xlsx`, скачивание и парсинг прогноза.
- [ ] Интеграция Python `bond_sync.py` с Go gateway.

Проверка: `go test ./...` и `go vet ./...` проходят.

---

## Файлы с внешними запросами (переносить в Go)

### 1. `app/services/moex.py` → MOEX ISS API — реализовано

**Класс:** `MoexService` (singleton: `moex_service = MoexService()`)
**HTTP клиент:** `httpx.AsyncClient`
**Базовый URL:** `settings.MOEX_ISS_URL`

#### Эндпоинты:
| Статус | Метод | URL | Назначение |
|---|---|---|---|
| [x] | GET | `{MOEX_ISS_URL}/engines/stock/markets/bonds/boards/TQCB/securities.json` | Весь рынок облигаций |
| [x] | GET | `{MOEX_ISS_URL}/securities/{isin}.json` | Описание бумаги |
| [x] | GET | `{MOEX_ISS_URL}/engines/stock/markets/bonds/securities/{isin}.json` | Рыночные данные |
| [x] | GET | `{MOEX_ISS_URL}/statistics/engines/stock/markets/bonds/bondization/{isin}.json` | Купоны / амортизация |

#### Логика:
- `get_bond_market_universe()` — фетчит весь список, затем конкурентный фан-аут через `asyncio.gather` + `asyncio.Semaphore(8)` на каждый ISIN
- `get_bond_full_info(isin)` — 3 последовательных HTTP-вызова на одну бумагу
- In-memory кэш: `self._cache` (по ISIN) + `self._market_cache` (весь рынок, TTL 15 мин)
- Таймаут universe fetch: 20.0 сек (hardcoded), per-bond: 10.0 сек (hardcoded)

#### Go-аналог:
- [x] `errgroup` + buffered channel как semaphore (размер 8)
- [x] Кэш реализован через Redis вместо локального `sync.Map` / `map` + `sync.RWMutex`
- [x] TTL market cache — 15 минут по умолчанию, настраивается через `MARKET_CACHE_TTL`
- [x] Парсинг description, marketdata, bondization и market universe
- [x] Сборка полной облигации из трёх последовательных запросов

---

### 2. `app/services/cbr_rates.py` → ЦБ РФ (scraping) — частично

**Класс:** `CbrRateService` (создаётся по требованию, не singleton)
**HTTP клиент:** `httpx.AsyncClient`
**Таймаут:** `settings.CBR_HTTP_TIMEOUT_SECONDS`

#### URLs (из settings):
| Setting | Назначение |
|---|---|
| `settings.CBR_KEY_RATE_URL` | HTML страница истории ключевой ставки |
| `settings.CBR_FORECAST_PAGE_URL` | HTML страница прогноза (парсится для поиска ссылки на .xlsx) |
| динамический | URL Excel-файла — находится в runtime из HTML прогноза |

#### Логика `get_snapshot()`:
1. Скачать HTML ключевой ставки → парсинг кастомным `HTMLParser` (`_TableParser`)
2. Скачать HTML прогноза → парсинг `_LinkParser` для нахождения ссылки на `.xlsx`
3. Скачать Excel `.xlsx` → парсинг через `openpyxl`

#### Go-аналог:
- [ ] HTML парсинг: `golang.org/x/net/html`
- [ ] Excel: `github.com/xuri/excelize/v2`
- [~] HTTP-клиент и методы `FetchKeyRatePage`, `FetchForecastPage`, `FetchForecastWorkbook` объявлены, но не реализованы
- [~] DTO `RateSnapshot`, `RatePoint`, `RateForecast` и HTTP endpoint объявлены, но приложение использует `StubService`

---

### 3. `app/services/bond_sync.py` — оркестрация (переносить частично) — не начато

Сам запросов не делает. Вызывает `moex_service` и пишет в БД через CRUD.

- [ ] Если Go — отдельный микросервис: `bond_sync.py` меняет только URL вызова (Python → Go HTTP/gRPC)
- [ ] Если полный перенос: переносить вместе с DB-логикой

---

## Что НЕ переносить

| Модуль | Причина |
|---|---|
| `app/api/` | FastAPI роуты — остаются на Python |
| `app/crud/` | DB операции — остаются на Python |
| `app/models/` | SQLAlchemy модели — остаются на Python |
| `app/db/` | Сессия БД — остаётся на Python |
| `app/core/` | Auth, config, security — остаётся на Python |

---

## Фактическая структура Go-модуля

```
moex-gateway/
├── go.mod
├── go.sum
├── cmd/gateway/main.go      # Точка входа HTTP-сервиса
├── internal/app/            # Сборка зависимостей и lifecycle
├── internal/httpserver/     # Fiber-роуты и middleware
├── internal/moex/           # Клиент, парсеры, сервис и типы MOEX
├── internal/cbr/            # Контракты и пока заглушки CBR
├── internal/cache/          # Redis-кэш
├── internal/config/         # Конфигурация из environment
└── docs/                    # Swagger
```

### Интеграция с Python:

- [ ] Python `bond_sync.py` вызывает Go-сервис через HTTP или gRPC.
- [ ] Меняется только URL/транспорт — логика оркестрации остаётся в Python.

---

## Зависимости Go

- [ ] `golang.org/x/net/html` — HTML парсинг CBR (`x/net` пока только косвенная зависимость)
- [ ] `github.com/xuri/excelize/v2` — Excel-парсинг CBR forecast
- [x] `golang.org/x/sync/errgroup` — конкурентный fan-out MOEX
- [x] `github.com/redis/go-redis/v9` — Redis-кэш MOEX
- [x] `github.com/gofiber/fiber/v3` — HTTP-сервис
