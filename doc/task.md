# Техническое задание: Mock-сервер Google Play Billing для тестирования In-App Purchases

**Версия документа:** 1.0  
**Дата:** Март 2026  
**Статус:** Черновик для согласования

---

## 1. Введение

### 1.1. Назначение документа
Настоящее техническое задание описывает требования к разработке Mock-сервера, имитирующего поведение Google Play Android Publisher API для тестирования серверной логики обработки внутриигровых покупок (In-App Purchases) и подписок в Android-приложениях.

### 1.2. Бизнес-цели
| Цель | Описание | Критерий успеха |
|------|----------|----------------|
| Ускорение разработки | Тестирование серверной логики без реальных транзакций | Сокращение цикла разработки на 30-50% |
| Повышение надёжности | Детерминированное поведение API для воспроизводимых тестов | 100% покрытие сценариев покупки/отмены/возврата |
| Снижение затрат | Исключение необходимости реальных платежей в тестовых средах | Нулевые расходы на тестовые транзакции |
| Безопасность | Тестирование без передачи реальных пользовательских данных | Соответствие GDPR/локальным нормам |

### 1.3. Область применения
- **Включает:** Эмуляция endpoints Google Play Developer API v3 для покупок, подписок, валидации токенов, управления редакциями
- **Не включает:** Реальную интеграцию с Google Play, обработку платежей, работу с финансовыми данными

---

## 2. Глоссарий и единый язык предметной области

| Термин | Определение |
|--------|-------------|
| **packageName** | Уникальный идентификатор Android-приложения (com.example.app) |
| **subscriptionId / productId** | Идентификатор товара или подписки в Google Play Console |
| **token / purchaseToken** | Уникальный токен покупки, выдаваемый Google после успешной оплаты |
| **editId** | Идентификатор сессии редактирования метаданных приложения |
| **acknowledge** | Подтверждение покупки сервером для завершения транзакции |
| **RTDN** | Real-time Developer Notifications — вебхуки от Google о изменениях подписок |
| **Aggregate** | Агрегат доменной модели — корень консистентности (Purchase, Subscription) |
| **Port** | Абстрактный интерфейс взаимодействия с внешним миром (репозиторий, API-клиент) |
| **Adapter** | Конкретная реализация порта для инфраструктуры (HTTP-клиент, БД) |

---

## 3. Границы системы и контексты взаимодействия

```
┌─────────────────────────────────────────────────────┐
│                 Backend-сервер приложения            │
├─────────────────────────────────────────────────────┤
│  • Бизнес-логика обработки покупок                  │
│  • Валидация токенов, управление доступом           │
│  • Отправка контента после подтверждения            │
└─────────────────┬───────────────────────────────────┘
                  │ HTTPS / JSON
                  ▼
┌─────────────────────────────────────────────────────┐
│              Mock-сервер Google Play API            │
│  (androidpublisher/v3/* endpoints)                  │
├─────────────────────────────────────────────────────┤
│  • purchases.subscriptions.get                      │
│  • purchases.products.get                           │
│  • orders:refund, subscriptions:cancel, etc.        │
│  • Эмуляция состояний: active, expired, canceled    │
└─────────────────┬───────────────────────────────────┘
                  │ (опционально)
                  ▼
┌─────────────────────────────────────────────────────┐
│          Реальный Google Play Developer API         │
│  (используется в production-среде)                  │
└─────────────────────────────────────────────────────┘
```

**Принцип переключения:** В runtime конфигурация определяет, какой implementation порта `GooglePublisherClient` используется: `RealGoogleClient` или `MockGoogleClient`.

---

## 4. Функциональные требования

### 4.1. Поддерживаемые эндпоинты (MVP)

| Метод | Endpoint | Описание | Mock-поведение |
|-------|----------|----------|----------------|
| GET | `/applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}` | Проверка статуса подписки | Возвращает детерминированный ответ на основе токена |
| GET | `/applications/{packageName}/purchases/products/{productId}/tokens/{token}` | Проверка разовой покупки | Аналогично подпискам |
| POST | `/applications/{packageName}/purchases/subscriptions/{token}:acknowledge` | Подтверждение покупки | Возвращает 200 OK, логирует событие |
| POST | `/applications/{packageName}/purchases/subscriptions/{token}:cancel` | Отмена подписки | Меняет состояние в mock-хранилище |
| POST | `/applications/{packageName}/purchases/subscriptions/{token}:refund` | Возврат средств | Эмулирует refund-сценарий |
| POST | `/applications/{packageName}/purchases/subscriptions/{token}:revoke` | Немедленная отмена доступа | Устанавливает статус revoked |
| POST | `/applications/{packageName}/purchases/subscriptions/{token}:defer` | Продление подписки | Обновляет expiryTimeMillis |

### 4.2. Управление состояниями Mock-данных

**Сценарии тестирования через конфигурацию токена:**

```json
{
  "token_scenarios": {
    "valid_active": {
      "acknowledgementState": "ACKNOWLEDGED",
      "autoRenewing": true,
      "expiryTimeMillis": "<future_timestamp>",
      "paymentState": 1,
      "purchaseState": 0
    },
    "expired_no_renew": {
      "autoRenewing": false,
      "expiryTimeMillis": "<past_timestamp>",
      "paymentState": 0
    },
    "canceled_user": {
      "cancelReason": 1,
      "userCancellationTimeMillis": "<timestamp>"
    },
    "pending_payment": {
      "paymentState": 2,
      "acknowledgementState": "PENDING"
    },
    "invalid_token": {
      "error": {
        "code": 404,
        "message": "Purchase token not found"
      }
    }
  }
}
```

### 4.3. Конфигурируемое поведение

- **Задержка ответа:** `mock_latency_ms: 0-5000` для тестирования таймаутов
- **Вероятность ошибки:** `error_rate: 0.0-1.0` для chaos-testing
- **Типы ошибок:** `5xx`, `4xx`, `network_timeout`, `malformed_json`
- **Версионирование ответов:** Поддержка разных версий схемы ответа Google API

---

## 5. Нефункциональные требования

### 5.1. Архитектурные принципы

```
┌─────────────────────────────────────┐
│         Presentation Layer          │
│  • HTTP Router (Gin/Echo/Fiber)    │
│  • Request/Response DTOs           │
│  • Validation, Logging, Metrics    │
└────────────────┬────────────────────┘
                 │ depends on abstractions
                 ▼
┌─────────────────────────────────────┐
│       Application Layer             │
│  • Use Cases / Application Services│
│  • Оркестрация бизнес-процессов    │
│  • Transaction boundaries          │
└────────────────┬────────────────────┘
                 │ depends on abstractions
                 ▼
┌─────────────────────────────────────┐
│          Domain Layer               │
│  • Entities: Purchase, Subscription│
│  • Value Objects: Token, Money     │
│  • Domain Services: Validation     │
│  • Domain Events: PurchaseConfirmed│
│  • Repository Interfaces (Ports)   │
└────────────────┬────────────────────┘
                 │ implements
                 ▼
┌─────────────────────────────────────┐
│       Infrastructure Layer          │
│  • MockRepository (in-memory/JSON) │
│  • RealGoogleClient (HTTP adapter) │
│  • Config Loader, Logger, Metrics  │
└─────────────────────────────────────┘
```

**Ключевые правила:**
- ✅ Domain-слой не зависит от Infrastructure, Frameworks, HTTP
- ✅ Все внешние зависимости объявлены как интерфейсы (Ports)
- ✅ Dependency Injection через конструкторы, без Service Locator
- ✅ Конфигурация вынесена в environment variables / config files

### 5.2. Технические требования

| Категория | Требование |
|-----------|-----------|
| **Язык/фреймворк** | Go 1.20+, стандартная библиотека + минимальные зависимости |
| **API контракт** | OpenAPI 3.0 спецификация, генерация кода/валидация |
| **Хранение состояний** | In-memory с возможностью persistence в JSON/Redis для интеграционных тестов |
| **Безопасность** | JWT/OAuth2 mock для заголовков авторизации, валидация service account |
| **Логирование** | Structured JSON logs, correlation ID, уровни: INFO/ERROR/DEBUG |
| **Метрики** | Prometheus-compatible: request_count, latency_histogram, error_rate |
| **Конфигурация** | Environment variables + YAML config, hot-reload для сценариев |
| **Тестируемость** | 90%+ покрытие unit-тестами, интеграционные тесты с реальным клиентом |

### 5.3. Требования к надёжности и эксплуатации

- **Идемпотентность:** Все mutating endpoints должны быть идемпотентными
- **Graceful degradation:** При ошибке конфигурации — fallback к deterministic default responses
- **Health checks:** `/health`, `/ready` endpoints для orchestrator (K8s)
- **Schema evolution:** Поддержка версионирования ответов, backward compatibility
- **Secrets management:** Токены и ключи — только через env vars / secret store, никогда в коде

---

## 6. Доменная модель (упрощённо)

### 6.1. Агрегаты и сущности

```go
// Domain Entities (упрощённое представление)

type PurchaseToken string  // Value Object

type SubscriptionPurchase struct {
    token              PurchaseToken
    productId          ProductID
    packageName        PackageName
    purchaseState      PurchaseState      // enum: purchased, canceled, pending
    paymentState       PaymentState       // enum: pending, approved, declined
    acknowledgementState AckState         // enum: pending, acknowledged
    expiryTimeMillis   int64              // Value Object: Timestamp
    autoRenewing       bool
    cancelReason       *CancelReason      // Optional Value Object
    // ... другие поля согласно Google API
}

// Invariants (гарантируемые методами агрегата):
// - token не может быть изменён после создания
// - expiryTimeMillis >= purchaseTimeMillis
// - acknowledgementState может перейти только pending → acknowledged
```

### 6.2. Domain Events

```go
type PurchaseAcknowledged struct {
    Token     PurchaseToken
    Timestamp time.Time
    UserID    string  // из контекста запроса
}

type SubscriptionExpired struct {
    Token     PurchaseToken
    ExpiryAt  time.Time
}

// Обработчики событий регистрируются в Application Layer,
// не в Domain — Domain только генерирует события.
```

### 6.3. Интерфейсы портов (Ports)

```go
// Port: репозиторий для хранения состояний покупок
type PurchaseRepository interface {
    GetByToken(ctx context.Context, token PurchaseToken) (*SubscriptionPurchase, error)
    Update(ctx context.Context, purchase *SubscriptionPurchase) error
    // Для mock: возможность предустановки сценариев
    SeedScenario(token PurchaseToken, scenario ScenarioConfig) error
}

// Port: клиент Google Publisher API (абстракция)
type GooglePublisherClient interface {
    GetSubscriptionPurchase(ctx context.Context, req GetSubscriptionRequest) (*SubscriptionPurchase, error)
    AcknowledgePurchase(ctx context.Context, req AcknowledgeRequest) error
    // ... другие методы
}

// Port: конфигурация и feature flags
type ConfigProvider interface {
    GetMockLatency() time.Duration
    GetErrorRate() float64
    IsFeatureEnabled(flag string) bool
}
```

---

## 7. API контракт (пример ответа)

### GET /applications/{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}

**Успешный ответ (200 OK):**
```json
{
  "kind": "androidpublisher#subscriptionPurchase",
  "acknowledgementState": 1,
  "autoRenewing": true,
  "expiryTimeMillis": "1735689600000",
  "paymentState": 1,
  "purchaseState": 0,
  "purchaseTimeMillis": "1704067200000",
  "orderId": "GPA.1234-5678-9012-34567",
  "productId": "premium_monthly",
  "regionCode": "RU",
  "cancelReason": null,
  "introductoryPriceInfo": null
}
```

**Ошибка (404 Not Found):**
```json
{
  "error": {
    "code": 404,
    "message": "Purchase token not found",
    "status": "NOT_FOUND"
  }
}
```

---

## 8. Стратегия тестирования

### 8.1. Уровни тестов

| Уровень | Что тестируем | Инструменты |
|---------|--------------|-------------|
| **Unit** | Domain-логика, invariants, value objects | `testing`, `testify` |
| **Integration** | Адаптеры: HTTP handler ↔ UseCase ↔ MockRepo | `httptest`, `testcontainers` |
| **Contract** | Соответствие ответов спецификации Google API | `openapi-validator`, `pact` |
| **E2E** | Сценарий: приложение → backend → mock → ответ | `k6`, `postman` |

### 8.2. Пример сценария интеграционного теста

```go
func TestSubscriptionAcknowledgement_Flow(t *testing.T) {
    // Arrange: инициализация mock-сервера с тестовым сценарием
    mockServer := NewMockServer(WithScenario("valid_pending"))
    defer mockServer.Close()
    
    // Act: вызов use-case через application service
    svc := NewPurchaseService(mockServer.Client(), mockLogger)
    err := svc.AcknowledgePurchase(ctx, "test_token_123")
    
    // Assert
    assert.NoError(t, err)
    assert.Eventually(t, func() bool {
        purchase, _ := mockServer.GetPurchase("test_token_123")
        return purchase.AcknowledgementState == Acknowledged
    }, 2*time.Second, 100*time.Millisecond)
    
    // Verify: событие опубликовано
    assert.Contains(t, mockLogger.Events(), "PurchaseAcknowledged")
}
```

### 8.3. Мониторинг актуальности Mock

**Cron-задача сравнения с реальным API:**
```yaml
# mock-sync-cron.yaml
schedule: "0 2 * * *"  # ежедневно в 02:00
job:
  - fetch_real_response: 
      endpoint: purchases.subscriptions.get
      token: "test_reference_token"
  - fetch_mock_response: same_endpoint
  - compare_schemas:
      ignore_fields: [orderId, purchaseTimeMillis]  # динамические
      fail_on: [missing_required_field, type_mismatch]
  - alert_if_drift: 
      channel: slack#backend-alerts
      severity: warning
```

---

## 9. Развёртывание и эксплуатация

### 9.1. Конфигурация среды

```bash
# .env.example
MOCK_SERVER_PORT=8080
LOG_LEVEL=info
METRICS_ENABLED=true

# Сценарии тестирования
MOCK_SCENARIOS_PATH=./config/scenarios.json
MOCK_DEFAULT_LATENCY_MS=100
MOCK_ERROR_RATE=0.0

# Безопасность (для staging)
AUTH_MOCK_ENABLED=true
MOCK_SERVICE_ACCOUNT_KEY_PATH=/secrets/mock-sa-key.json

# Интеграция с реальным API (fallback)
GOOGLE_API_ENABLED=false  # true только в production
GOOGLE_API_CREDENTIALS_PATH=/secrets/google-sa-key.json
```

### 9.2. Docker-образ

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o mock-server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mock-server .
COPY config/ ./config/
EXPOSE 8080
ENTRYPOINT ["./mock-server"]
```



---

## 10. План реализации (Roadmap)

| Этап | Срок | Результат |
|------|------|-----------|
| **Phase 0: Подготовка** | Неделя 1 | OpenAPI spec, domain model review, выбор стека |
| **Phase 1: Ядро** | Недели 2-3 | Domain layer, ports, mock repository, базовый HTTP server |
| **Phase 2: Endpoints MVP** | Недели 4-5 | Реализация 5 ключевых endpoints, конфигурация сценариев |
| **Phase 3: Инфраструктура** | Неделя 6 | Logging, metrics, config management, Docker/K8s manifests |
| **Phase 4: Тестирование** | Неделя 7 | Unit/integration tests, contract tests, E2E сценарии |
| **Phase 5: Документация** | Неделя 8 | Developer guide, API docs, runbook для эксплуатации |
| **Phase 6: Pilot** | Неделя 9 | Развёртывание в staging, onboarding 1-2 команд |
| **Phase 7: Production** | Неделя 10+ | Мониторинг, обратная связь, итеративное улучшение |

---

## 11. Критерии приёмки

- [ ] Все endpoints из раздела 4.1 возвращают валидные JSON-ответы согласно спецификации Google API
- [ ] Переключение между mock/real реализациями происходит через конфигурацию без изменения кода приложения
- [ ] Покрытие unit-тестами domain-слоя ≥ 90%, интеграционными — ≥ 70%
- [ ] Mock-сервер обрабатывает 100 RPS с p95 latency < 50ms на тестовом окружении
- [ ] Конфигурация сценариев позволяет воспроизвести ≥ 10 различных состояний покупки/подписки
- [ ] Документация позволяет новому разработчику запустить сервер и написать тест за < 30 минут
- [ ] Реализован механизм уведомления о расхождении с реальным API (cron-sync)

---

## 12. Риски и меры mitigation

| Риск | Вероятность | Влияние | Мера снижения |
|------|-------------|---------|---------------|
| Google изменит формат ответа API | Средняя | Высокое | Cron-сравнение схем, версионирование mock-ответов |
| Mock-сервер станет "god object" | Высокая | Среднее | Строгое следование SRP, code review с фокусом на архитектуру |
| Утечка mock-эндпоинтов в production | Низкая | Критическое | Feature flags, separate deployment |
| Недостаточная реалистичность mock | Средняя | Среднее | Регулярная синхронизация с real API, участие QA в дизайне сценариев |

---

## 13. Приложения

### A. Ссылки на документацию
- [Google Play Developer API Reference](https://developers.google.com/android-publisher/api-ref)
- [Android Publisher Go Client Library](https://pkg.go.dev/google.golang.org/api/androidpublisher/v3)
- [Hexagonal Architecture (Ports & Adapters)](https://alistair.cockburn.us/hexagonal-architecture/)
- [Domain-Driven Design Reference](https://domainlanguage.com/ddd/reference/)

### B. Шаблон ADR (Architecture Decision Record)
```markdown
# ADR-001: Выбор языка и фреймворка для Mock-сервера

## Статус
Принято

## Контекст
Требуется лёгкий, высокопроизводительный сервер с минимальными зависимостями...

## Решение
Go + стандартная библиотека + chi для роутинга...

## Последствия
+ Быстрая компиляция, статическая линковка
+ Простота деплоя (один бинарник)
- Меньше готовых middleware vs Node.js/Python
```

---

> **Примечание для команды:** Данный документ является living document. Все изменения архитектуры должны сопровождаться обновлением ADR. Приоритет — простота, явность контрактов и возможность быстрой итерации.

# Технологический стек на Go для Mock-сервера Google Play Billing

## 🎯 Принципы выбора стека
- **Минимализм**: меньше зависимостей → проще поддержка и безопасность
- **Явность**: предпочитать стандартную библиотеку там, где это не усложняет код
- **Тестируемость**: все компоненты должны легко мокироваться
- **Наблюдаемость**: встроенная поддержка метрик, логов, трассировки
- **Совместимость**: Go 1.20+ для стабильности и долгосрочной поддержки

---

## 📦 Базовый стек

### Язык и инструментация
```yaml
go_version: "1.21+"
build_tool: "go build (CGO_ENABLED=0 для статической линковки)"
module_manager: "Go modules (go.mod)"
formatter: "gofmt + goimports"
linter: "golangci-lint"
vendor: "не используется (зависимости через go.mod)"
```

### Веб-фреймворк и роутинг
```go
// Рекомендуется: chi (легковесный, idiomatic, совместим с stdlib)
github.com/go-chi/chi/v5          // роутинг с middleware support
github.com/go-chi/chi/v5/middleware // готовый middleware: logger, recover, timeout

// Альтернативы (по необходимости):
// - gin-gonic/gin: больше фич, но тяжелее
// - fiber: быстрый, но требует адаптации под stdlib интерфейсы
// - стандартный http.ServeMux: если нужна максимальная простота
```

### Обработка JSON и валидация
```go
// Стандартная библиотека для большинства случаев
encoding/json                     // сериализация/десериализация

// Для строгой валидации структур:
github.com/go-playground/validator/v10  // теги валидации, кастомные правила

// Для генерации DTO из OpenAPI:
github.com/deepmap/oapi-codegen   // codegen из OpenAPI 3.0 спецификации
github.com/getkin/kin-openapi     // runtime валидация запросов/ответов

// Опционально: fast JSON для high-load сценариев
github.com/json-iterator/go       // drop-in замена encoding/json
```

### Конфигурация и feature flags
```go
// Чтение конфигурации:
github.com/spf13/viper            // env vars, YAML, JSON, hot-reload
github.com/spf13/pflag            // CLI flags для локального запуска

// Управление секретами:
// - env vars в development
// - Kubernetes Secrets / HashiCorp Vault в production

// Feature flags (опционально):
github.com/thomasjtaylor/fflags   // простой in-memory flags manager
// или интеграция с LaunchDarkly/Unleash при необходимости
```

### Логирование и наблюдаемость
```go
// Structured logging:
go.uber.org/zap                   // быстрый, типобезопасный, structured JSON logs
// или
github.com/rs/zerolog             // компактный JSON, удобный API

// Метрики:
github.com/prometheus/client_golang  // Prometheus metrics exporter

// Трассировка (опционально):
go.opentelemetry.io/otel          // OpenTelemetry для distributed tracing
go.opentelemetry.io/otel/exporters/otlp/otlptrace

// Health checks:
// Реализуется на chi middleware + custom endpoints /health, /ready
```

### Тестирование
```go
// Unit-тесты:
testing                           // стандартная библиотека
github.com/stretchr/testify       // assert/require, mock-генерация

// Mock-генерация:
github.com/golang/mock/mockgen    // генерация mock-имплементаций интерфейсов
// или
github.com/vektra/mockery/v2      // более современный аналог

// Интеграционные тесты:
net/http/httptest                 // тестовый HTTP-сервер
github.com/testcontainers/testcontainers-go  // Docker-контейнеры для тестов

// Нагрузочное тестирование:
// k6 (внешний инструмент) или vegeta для API-тестов

// Покрытие:
go test -coverprofile=coverage.out
github.com/wadey/gocovmerge       // агрегация покрытия из нескольких пакетов
```

### Dependency Injection
```go
// Рекомендуемый подход: ручная композиция через конструкторы
// (простота, явность, тестируемость)

// Опционально: легковесный DI-контейнер для сложных графов
github.com/google/wire          // compile-time DI, codegen
// или
go.uber.org/dig                 // runtime reflection-based DI

// Избегать: service locator, глобальные синглтоны
```

### Работа с данными и состоянием
```go
// In-memory хранилище для mock-сценариев:
// - sync.Map для конкурентного доступа
// - map[token]Scenario + mutex для простоты

// Персистентность (опционально для интеграционных тестов):
github.com/allegro/bigcache     // быстрый in-memory cache с TTL
// или
github.com/redis/go-redis/v9    // Redis для shared state между инстансами

// Генерация тестовых данных:
github.com/brianvoe/gofakeit/v6  // fake data generation (имитация faker.js)
```

### API документация и контракт
```go
// OpenAPI 3.0 спецификация:
// - swagger.yaml / openapi.yaml в корне репозитория
// - валидация через github.com/getkin/kin-openapi

// Автогенерация документации:
github.com/swaggo/swag          // аннотации в коде → Swagger UI
github.com/swaggo/http-swagger  // middleware для Swagger UI

// Контрактное тестирование (опционально):
github.com/pact-foundation/pact-go  // consumer-driven contracts
```

### Утилиты и вспомогательные пакеты
```go
// Работа с временем и таймаутами:
github.com/avast/retry-go/v4    // retry logic с backoff
github.com/sethvargo/go-retry   // простой retry API

// Обработка ошибок:
github.com/pkg/errors           // stack traces (в Go 1.13+ можно использовать stdlib errors)

// Контекст и отмена операций:
context                         // стандартная библиотека

// Парсинг и форматирование:
github.com/google/uuid          // UUID generation
github.com/shopspring/decimal   // точные денежные расчёты (если нужно)
```

---

## 🗂️ Структура проекта (Clean/Hexagonal Architecture)

```
/cmd/server/
  └── main.go                   # точка входа, композиция зависимостей

/internal/
  ├── domain/                   # Domain Layer (чистая бизнес-логика)
  │   ├── entity/              # Purchase, Subscription, Token
  │   ├── valueobject/         # Money, Timestamp, PackageName
  │   ├── repository/          # интерфейсы портов (PurchaseRepository)
  │   ├── service/             # Domain Services (валидации)
  │   └── event/               # Domain Events
  │
  ├── application/             # Application Layer
  │   ├── usecase/             # Use Cases (AcknowledgePurchase, GetSubscription)
  │   ├── port/                # входные порты (HTTP handlers interfaces)
  │   └── dto/                 # Request/Response DTOs
  │
  ├── infrastructure/          # Infrastructure Layer
  │   ├── http/                # chi router, middleware, handlers
  │   ├── mock/                # MockRepository, ScenarioManager
  │   ├── config/              # Viper config loader
  │   ├── logger/              # Zap/Zerolog wrapper
  │   ├── metrics/             # Prometheus setup
  │   └── persistence/         # (опционально) Redis/JSON storage
  │
  └── pkg/                     # переиспользуемые утилиты
      ├── errorx/              # типизированные ошибки приложения
      ├── validator/           # кастомные валидаторы
      └── testutil/            # общие утилиты для тестов

/api/
  ├── openapi.yaml             # OpenAPI 3.0 спецификация
  └── contracts/               # JSON schemas для валидации

/config/
  ├── default.yaml             # конфигурация по умолчанию
  ├── scenarios/               # predefined mock scenarios
  └── test/                    # конфигурация для тестов

/deploy/
  ├── Dockerfile
  ├── docker-compose.yml
  └── k8s/                     # Kubernetes manifests

/scripts/
  ├── generate-mocks.sh        # mockgen wrapper
  ├── validate-openapi.sh      # проверка спецификации
  └── run-e2e.sh               # запуск интеграционных тестов
```

---

## ⚙️ Конфигурация запуска

### go.mod (фрагмент)
```go
module github.com/bivex/google-billing-mock

go 1.21

require (
    github.com/go-chi/chi/v5 v5.0.11
    github.com/spf13/viper v1.18.2
    go.uber.org/zap v1.26.0
    github.com/prometheus/client_golang v1.18.0
    github.com/stretchr/testify v1.8.4
    github.com/go-playground/validator/v10 v10.16.0
    github.com/getkin/kin-openapi v0.124.0
    github.com/brianvoe/gofakeit/v6 v6.28.0
)
```

### Makefile (основные цели)
```makefile
.PHONY: build test lint mock-gen run

build:
	CGO_ENABLED=0 go build -o bin/mock-server ./cmd/server

test:
	go test -race -coverprofile=coverage.out ./...

test-e2e:
	go test -tags=e2e ./integration/...

lint:
	golangci-lint run ./...

mock-gen:
	mockgen -source=internal/domain/repository/purchase.go \
		-destination=internal/infrastructure/mock/purchase_mock.go \
		-package=mock

run:
	go run ./cmd/server --config=config/default.yaml

docker-build:
	docker build -t google-billing-mock:latest .

openapi-gen:
	oapi-codegen -generate types,server,spec \
		-package=api ./api/openapi.yaml > internal/api/generated.go
```

---

## 🧪 Стратегия тестирования

| Тип теста | Инструменты | Покрытие | Скорость |
|-----------|-------------|----------|----------|
| **Unit** | `testing`, `testify`, `mockgen` | Domain: ≥90% | <100ms/test |
| **Integration** | `httptest`, `testcontainers` | Adapters: ≥70% | <500ms/test |
| **Contract** | `kin-openapi`, `pact-go` | API contracts: 100% | <1s/test |
| **E2E** | `k6`, custom Go client | Критические сценарии | Ручной запуск |
| **Chaos** | `avast/retry-go`, custom middleware | Resilience paths | По требованию |

### Пример mock-генерации
```bash
# Генерация mock для PurchaseRepository
mockgen -source=internal/domain/repository/purchase.go \
  -destination=internal/infrastructure/mock/purchase_mock.go \
  -package=mock \
  -copyright_file=scripts/copyright.txt

# Использование в тесте
ctrl := gomock.NewController(t)
defer ctrl.Finish()
mockRepo := mock.NewMockPurchaseRepository(ctrl)
mockRepo.EXPECT().GetByToken(gomock.Any(), "test_token").Return(validPurchase, nil)
```

---

## 🚀 Сборка и деплой

### Dockerfile (multi-stage)
```dockerfile
# Stage 1: Build
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o mock-server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/mock-server .
COPY config/ ./config/
COPY api/openapi.yaml ./api/

EXPOSE 8080
USER nobody:nobody
ENTRYPOINT ["./mock-server"]
```

### Kubernetes readiness (фрагмент)
```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
    httpHeaders:
    - name: X-Readiness-Check
      value: "mock-server"
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 3

livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 30
```

---

## 🔐 Безопасность и best practices

```go
// 1. Никогда не логировать токены и PII
zap.L().Debug("Purchase request", 
    zap.String("package", pkg), 
    zap.String("product_id", pid)
    // ❌ zap.String("token", token) — никогда!
)

// 2. Валидация входных данных на границе
if err := validator.Struct(req); err != nil {
    return http.StatusBadRequest, errors.Wrap(err, "invalid request")
}

// 3. Контекст с таймаутом для всех внешних вызовов
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()

// 4. Rate limiting для предотвращения abuse
middleware.Throttle(100 * time.Millisecond) // 10 RPS per IP

// 5. Secrets только через env vars
config.NewFromEnv() // не читать из файлов в коде
```

---

## 📊 Метрики для мониторинга (Prometheus)

```go
var (
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "mock_server_request_duration_seconds",
            Help:    "Request latency distribution",
            Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
        },
        []string{"method", "endpoint", "status"},
    )
    
    mockScenarioHits = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mock_scenario_executions_total",
            Help: "Number of times each mock scenario was triggered",
        },
        []string{"scenario_name", "endpoint"},
    )
    
    apiDriftAlerts = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "mock_api_drift_alerts_total",
            Help: "Number of times mock response drifted from real API",
        },
    )
)
```

---

## 🔄 CI/CD интеграция (GitHub Actions пример)

```yaml
# .github/workflows/test.yml
name: Test & Build
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with: { go-version: '1.21' }
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    
    - name: Lint
      run: make lint
    
    - name: Test
      run: make test
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with: { file: ./coverage.out }
    
    - name: Build
      run: make build
    
    - name: Docker build
      run: make docker-build
```

---

> 💡 **Рекомендация**: Начните с минимального стека (`chi + stdlib + zap + viper`), добавляйте зависимости только при обоснованной необходимости. Каждая новая библиотека — это технический долг на поддержку и безопасность.

Этот стек обеспечивает баланс между производительностью, поддерживаемостью и скоростью разработки, полностью соответствуя архитектурным принципам из технического задания.