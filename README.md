# grinex-adapter-service

gRPC-сервис, который отдаёт курс USDT/A7A5 по стакану биржи Grinex.
Курс считается одним из двух методов, каждый результат сохраняется в PostgreSQL.

## Методы расчёта


| Метод          | Параметры | Что считает                                                          |
| ------------------- | ------------------ | ------------------------------------------------------------------------------ |
| `CALCMETHOD_TOP_N`  | `n`                | цена уровня`n` (с 1) в asks и bids                                |
| `CALCMETHOD_AVG_NM` | `n`, `m`           | средняя цена уровней с`n` по `m` включительно |

Уровни отсортированы от лучшего: asks по возрастанию цены, bids по убыванию.

Ошибки:


| Ситуация                                                                                   | gRPC code                                          |
| -------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| неизвестный метод,`n < 1`, `n > m`, `n`/`m` за пределами стакана | `InvalidArgument`                                  |
| биржа недоступна (таймаут, не-200, не-JSON)                              | `Unavailable`                                      |
| прочее, например ошибка БД                                                   | `Internal` (детали только в логе) |

## Быстрый старт

Нужны Docker и [grpcurl](https://github.com/fullstorydev/grpcurl).

```bash
docker compose up -d --build
docker compose ps        # у app должен быть статус (healthy)
```

Запросы:

```bash
grpcurl -plaintext -d '{"calc_method":"CALCMETHOD_TOP_N","n":1}' \
  localhost:50051 rates.v1.RatesService/GetRates

grpcurl -plaintext -d '{"calc_method":"CALCMETHOD_AVG_NM","n":1,"m":3}' \
  localhost:50051 rates.v1.RatesService/GetRates

grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

Ответ:

```json
{
  "ask": 80.15,
  "bid": 80.05,
  "fetchedAt": "2026-11-02T12:28:01.555083849Z"
}
```

Сохранённые курсы:

```bash
docker compose exec postgres psql -U rates -d rates -c "SELECT * FROM rates ORDER BY id DESC LIMIT 5;"
```

## Локальный запуск без контейнера приложения

```bash
docker compose up -d postgres jaeger
GRINEX_USE_STUB=true make run
```

## Конфигурация

Приоритет: **флаг > переменная окружения > значение по умолчанию**.


| Env                           | Флаг          | По умолчанию                                       | Описание                                                          |
| ----------------------------- | ----------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `GRPC_ADDR`                   | `-port`           | `:50051`                                                      | адрес gRPC-сервера                                            |
| `METRICS_ADDR`                | `-metrics-addr`   | `:9090`                                                       | адрес HTTP`/metrics`                                                 |
| `DATABASE_DSN`                | `-dsn`            | `postgres://rates:rates@localhost:5432/rates?sslmode=disable` | строка подключения к PostgreSQL                         |
| `APP_TIMEOUT`                 | `-app-timeout`    | `10s`                                                         | таймаут graceful shutdown                                          |
| `GRINEX_BASE_URL`             | `-grinex-url`     | `https://grinex.io`                                           | базовый URL биржи                                             |
| `GRINEX_MARKET`               | —                | `usdta7a5`                                                    | торговая пара                                                 |
| `GRINEX_TIMEOUT`              | `-grinex-timeout` | `5s`                                                          | таймаут запроса к бирже                               |
| `GRINEX_USE_STUB`             | —                | `false`                                                       | `true` — фиксированный стакан вместо биржи |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | —                | `localhost:4317`                                              | OTLP gRPC endpoint для трейсов                                  |

Пример: `go run ./cmd -port :6000 -grinex-timeout 2s`.

## Наблюдаемость

- **Логи**: JSON (zap) в stdout, по строке на каждый gRPC-вызов: метод, код, длительность.
- **Трейсы**: OpenTelemetry → Jaeger, UI на http://localhost:16686. Спаны: gRPC-вызов → use case → HTTP к бирже → `postgres.Save`.
- **Метрики**: Prometheus на http://localhost:9090/metrics:
  - `grpc_server_requests_total{method,code}`, `grpc_server_request_duration_seconds{method}`
  - `exchange_requests_total{result}`, `exchange_request_duration_seconds`
- **Health**: стандартный `grpc.health.v1`. При остановке сервис отвечает `NOT_SERVING`, затем выполняется `GracefulStop`.

## Про API Grinex

На момент разработки публичный API Grinex был недоступен: `/api/v1/...` отвечал 404, `/api/v2/depth` — 403 от WAF.
Клиент написан под формат Garantex-совместимого `GET /api/v2/depth?market=...`. Это **предположение**, формат зафиксирован
в фикстуре `internal/adapter/web/rest/grinex/testdata/depth.json` и проверяется тестами.
Поэтому в `docker-compose.yaml` стоит `GRINEX_USE_STUB=true`. Чтобы обращаться к настоящей бирже, поставьте `false`.

## Разработка


| Команда      | Что делает                                   |
| ------------------- | ----------------------------------------------------- |
| `make build`        | статический бинарник в`./bin/app` |
| `make test`         | unit-тесты                                       |
| `make lint`         | golangci-lint                                         |
| `make run`          | запуск локально                         |
| `make docker-build` | сборка образа                             |
| `make generate`     | генерация gRPC-кода из proto (buf)     |

## Структура

```
api/rates/v1/          proto-контракт
gen/                   сгенерированный код (коммитится)
cmd/                   main: сборка зависимостей, запуск, graceful shutdown
internal/domain/       стакан, TopN/AvgNM, ошибки
internal/application/  use case GetRates и порты
internal/adapter/      gRPC-хендлер, клиент Grinex, PostgreSQL
internal/config/       env + флаги
internal/metrics/      Prometheus
internal/telemetry/    OpenTelemetry
migrations/            SQL-миграции (goose, встроены через embed)
```
