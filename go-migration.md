# Go Migration: myinvesthelper_backend external requests

## Источник: FastAPI бэкенд
Path: `/Users/i.chutchev/Pycharm Projects/personal/myinvesthelper_backend`

---

## Файлы с внешними запросами (переносить в Go)

### 1. `app/services/moex.py` → MOEX ISS API

**Класс:** `MoexService` (singleton: `moex_service = MoexService()`)
**HTTP клиент:** `httpx.AsyncClient`
**Базовый URL:** `settings.MOEX_ISS_URL`

#### Эндпоинты:
| Метод | URL | Назначение |
|---|---|---|
| GET | `{MOEX_ISS_URL}/engines/stock/markets/bonds/boards/TQCB/securities.json` | Весь рынок облигаций |
| GET | `{MOEX_ISS_URL}/securities/{isin}.json` | Описание бумаги |
| GET | `{MOEX_ISS_URL}/engines/stock/markets/bonds/securities/{isin}.json` | Рыночные данные |
| GET | `{MOEX_ISS_URL}/statistics/engines/stock/markets/bonds/bondization/{isin}.json` | Купоны / амортизация |

#### Логика:
- `get_bond_market_universe()` — фетчит весь список, затем конкурентный фан-аут через `asyncio.gather` + `asyncio.Semaphore(8)` на каждый ISIN
- `get_bond_full_info(isin)` — 3 последовательных HTTP-вызова на одну бумагу
- In-memory кэш: `self._cache` (по ISIN) + `self._market_cache` (весь рынок, TTL 15 мин)
- Таймаут universe fetch: 20.0 сек (hardcoded), per-bond: 10.0 сек (hardcoded)

#### Go-аналог:
- `errgroup` + buffered channel как semaphore (размер 8)
- `sync.Map` или `map` + `sync.RWMutex` для кэша
- `time.Now().Add(15 * time.Minute)` для TTL market cache

---

### 2. `app/services/cbr_rates.py` → ЦБ РФ (scraping)

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
- HTML парсинг: `golang.org/x/net/html`
- Excel: `github.com/xuri/excelize/v2`
- Приватные хелперы `_fetch_text()` / `_fetch_bytes()` → простые Go-функции с `http.Client`

---

### 3. `app/services/bond_sync.py` — оркестрация (переносить частично)

Сам запросов не делает. Вызывает `moex_service` и пишет в БД через CRUD.

- Если Go — отдельный микросервис: `bond_sync.py` меняет только URL вызова (Python → Go HTTP/gRPC)
- Если полный перенос: переносить вместе с DB-логикой

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

## Предлагаемая структура Go-модуля

```
go-moex-cbr/
├── go.mod
├── go.sum
├── main.go                  # HTTP сервер (или gRPC), вызываемый из Python
├── moex/
│   ├── client.go            # HTTP-клиент + in-memory кэш
│   └── types.go             # Структуры JSON-ответов MOEX ISS
└── cbr/
    ├── client.go            # HTTP-клиент + HTML/Excel парсинг
    └── types.go             # Структуры данных ЦБ РФ
```

### Интеграция с Python:
Python `bond_sync.py` вызывает Go-сервис через HTTP или gRPC.
Меняется только URL/транспорт — логика оркестрации остаётся в Python.

---

## Зависимости Go

```
golang.org/x/net/html          # HTML парсинг (CBR)
github.com/xuri/excelize/v2    # Excel парсинг (CBR forecast)
golang.org/x/sync/errgroup     # Конкурентный фан-аут (MOEX)
```
