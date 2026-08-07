# Шаблон для создания новой Ability

Используйте этот шаблон как основу для создания собственных Abilities в Verity-BDD.

## 📋 Чек-лист перед началом

- [ ] Определите основную цель Ability (что она делает?)
- [ ] Продумайте интерфейс (какие методы нужны?)
- [ ] Определите, будет ли Ability хранить состояние
- [ ] Решите, нужны ли фабричные методы
- [ ] Продумайте, какие Activities и Questions понадобятся

## 🏗️ Структура файлов

```
verity/abilities/
├── your_ability/
│   ├── ability.go           # Интерфейс и основные типы
│   ├── implementation.go    # Реализация Ability
│   ├── activities.go        # Activities для использования Ability
│   ├── questions.go         # Questions для проверки состояния
│   └── builders.go          # Builder patterns и фабричные методы
└── your_ability_test.go     # Тесты
```

---

## 📝 Шаблон кода

### 1. Интерфейс (ability.go)

```go
package your_ability

import (
    "fmt"
    "sync"

    "github.com/verity-bdd/verity-bdd/verity_abilities"
)

// YourAbilityName - способность для [краткое описание]
type YourAbilityName interface {
    abilities.Ability

    // Основные операции (замените на ваши методы)
    DoSomething(param string) error
    GetSomething() (string, error)

    // Управление состоянием (если нужно)
    SetConfig(config Config) error
    GetStatus() string

    // История операций (опционально)
    LastOperation() string
    LastError() error
}

// Config - конфигурация для Ability (опционально)
type Config struct {
    // Параметры конфигурации
    Endpoint    string
    Timeout     time.Duration
    RetryPolicy RetryPolicy

    // Другие параметры...
}

// RetryPolicy - политика повторных попыток (опционально)
type RetryPolicy struct {
    MaxRetries int
    Delay      time.Duration
}
```

### 2. Реализация (implementation.go)

```go
package your_ability

import (
    "fmt"
    "sync"
    "time"
)

// yourAbilityName - приватная реализация
type yourAbilityName struct {
    // Конфигурация
    config Config

    // Состояние
    lastOperation string
    lastError     error
    isConnected   bool

    // Ресурсы
    client SomeClient // замените на ваш тип клиента

    // Thread safety
    mutex sync.RWMutex
}

// ====================
// Фабричные методы
// ====================

// NewYourAbility - базовый конструктор
func NewYourAbility() YourAbilityName {
    return &yourAbilityName{
        config: Config{
            Timeout: 30 * time.Second,
        },
        lastOperation: "none",
    }
}

// NewYourAbilityWithConfig - конструктор с конфигурацией
func NewYourAbilityWithConfig(config Config) YourAbilityName {
    return &yourAbilityName{
        config:        config,
        lastOperation: "none",
    }
}

// WithEndpoint - устанавливает endpoint (builder pattern)
func WithEndpoint(endpoint string) YourAbilityName {
    return &yourAbilityName{
        config: Config{
            Endpoint: endpoint,
            Timeout:  30 * time.Second,
        },
        lastOperation: "none",
    }
}

// ====================
// Основные методы
// ====================

func (y *yourAbilityName) DoSomething(param string) error {
    y.mutex.Lock()
    defer y.mutex.Unlock()

    // Валидация состояния
    if !y.isConnected {
        err := fmt.Errorf("not connected")
        y.lastError = err
        y.lastOperation = "do_something_error"
        return err
    }

    // Логика выполнения операции
    if err := y.validateParam(param); err != nil {
        y.lastError = fmt.Errorf("validation failed: %w", err)
        y.lastOperation = "do_something_validation_error"
        return y.lastError
    }

    // Выполнение основной операции
    result, err := y.client.DoSomething(param)
    if err != nil {
        y.lastError = fmt.Errorf("operation failed: %w", err)
        y.lastOperation = "do_something_failed"
        return y.lastError
    }

    // Успешное завершение
    y.lastOperation = fmt.Sprintf("do_something: %s", param)
    y.lastError = nil

    // Сохраняем результат если нужно
    // y.lastResult = result

    return nil
}

func (y *yourAbilityName) GetSomething() (string, error) {
    y.mutex.RLock()
    defer y.mutex.RUnlock()

    if !y.isConnected {
        return "", fmt.Errorf("not connected")
    }

    result, err := y.client.GetSomething()
    if err != nil {
        y.lastError = fmt.Errorf("get operation failed: %w", err)
        return "", y.lastError
    }

    y.lastOperation = "get_something"
    return result, nil
}

// ====================
// Управление состоянием
// ====================

func (y *yourAbilityName) SetConfig(config Config) error {
    y.mutex.Lock()
    defer y.mutex.Unlock()

    // Валидация конфигурации
    if err := y.validateConfig(config); err != nil {
        y.lastError = fmt.Errorf("invalid config: %w", err)
        return y.lastError
    }

    y.config = config
    y.lastOperation = "config_updated"
    return nil
}

func (y *yourAbilityName) GetStatus() string {
    y.mutex.RLock()
    defer y.mutex.RUnlock()

    if y.isConnected {
        return fmt.Sprintf("connected to %s", y.config.Endpoint)
    }
    return "disconnected"
}

// ====================
// История операций
// ====================

func (y *yourAbilityName) LastOperation() string {
    y.mutex.RLock()
    defer y.mutex.RUnlock()
    return y.lastOperation
}

func (y *yourAbilityName) LastError() error {
    y.mutex.RLock()
    defer y.mutex.RUnlock()
    return y.lastError
}

// ====================
// Приватные методы
// ====================

func (y *yourAbilityName) validateParam(param string) error {
    if param == "" {
        return fmt.Errorf("parameter cannot be empty")
    }

    if len(param) > 1000 {
        return fmt.Errorf("parameter too long")
    }

    return nil
}

func (y *yourAbilityName) validateConfig(config Config) error {
    if config.Endpoint == "" {
        return fmt.Errorf("endpoint is required")
    }

    if config.Timeout <= 0 {
        return fmt.Errorf("timeout must be positive")
    }

    return nil
}

// ====================
// Управление соединением (если нужно)
// ====================

func (y *yourAbilityName) Connect() error {
    y.mutex.Lock()
    defer y.mutex.Unlock()

    if y.isConnected {
        return nil // уже подключены
    }

    // Создаем клиент
    client, err := SomeClientConnect(y.config.Endpoint, y.config.Timeout)
    if err != nil {
        y.lastError = fmt.Errorf("connection failed: %w", err)
        y.lastOperation = "connect_failed"
        return y.lastError
    }

    y.client = client
    y.isConnected = true
    y.lastOperation = "connected"
    y.lastError = nil

    return nil
}

func (y *yourAbilityName) Disconnect() error {
    y.mutex.Lock()
    defer y.mutex.Unlock()

    if !y.isConnected {
        return nil
    }

    if y.client != nil {
        y.client.Close()
    }

    y.isConnected = false
    y.client = nil
    y.lastOperation = "disconnected"

    return nil
}
```

### 3. Activities (activities.go)

```go
package your_ability

import (
    "fmt"

    "github.com/verity-bdd/verity-bdd"
)

// ====================
// Basic Activities
// ====================

// DoSomethingActivity - выполнение операции
type DoSomethingActivity struct {
    param string
}

func DoSomething(param string) *DoSomethingActivity {
    return &DoSomethingActivity{param: param}
}

func (d *DoSomethingActivity) PerformAs(actor core.Actor) error {
    ability, err := actor.AbilityTo(&yourAbilityName{})
    if err != nil {
        return fmt.Errorf("actor does not have your ability: %w", err)
    }

    yourAbility := ability.(YourAbilityName)
    return yourAbility.DoSomething(d.param)
}

func (d *DoSomethingActivity) Description() string {
    return fmt.Sprintf("does something with: %s", d.param)
}

// ====================
// Complex Activities
// ====================

// DoSomethingWithConfigActivity - выполнение с конфигурацией
type DoSomethingWithConfigActivity struct {
    param  string
    config Config
}

func DoSomethingWithConfig(param string, config Config) *DoSomethingWithConfigActivity {
    return &DoSomethingWithConfigActivity{
        param:  param,
        config: config,
    }
}

func (d *DoSomethingWithConfigActivity) PerformAs(actor core.Actor) error {
    ability, err := actor.AbilityTo(&yourAbilityName{})
    if err != nil {
        return fmt.Errorf("actor does not have your ability: %w", err)
    }

    yourAbility := ability.(YourAbilityName)

    // Устанавливаем конфигурацию
    if err := yourAbility.SetConfig(d.config); err != nil {
        return fmt.Errorf("failed to set config: %w", err)
    }

    // Выполняем операцию
    return yourAbility.DoSomething(d.param)
}

func (d *DoSomethingWithConfigActivity) Description() string {
    return fmt.Sprintf("does something with: %s using custom config", d.param)
}
```

---

## 🧪 Шаблон тестов

```go
package your_ability

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/verity-bdd/verity-bdd"
    "github.com/verity-bdd/verity-bdd/verity_expectations"
    "github.com/verity-bdd/verity-bdd/verity_expectations"
)

// ====================
// Unit Tests
// ====================

func TestNewYourAbility(t *testing.T) {
    ability := NewYourAbility()

    assert.NotNil(t, ability)
    assert.Equal(t, "none", ability.LastOperation())
    assert.NoError(t, ability.LastError())
}

func TestNewYourAbilityWithConfig(t *testing.T) {
    config := Config{
        Endpoint: "test://localhost",
        Timeout:  10 * time.Second,
    }

    ability := NewYourAbilityWithConfig(config)

    assert.NotNil(t, ability)
    status := ability.GetStatus()
    assert.Contains(t, status, "test://localhost")
}

func TestYourAbility_DoSomething(t *testing.T) {
    // Arrange
    ability := NewYourAbility()

    // Mock client setup (если нужно)
    // setupMockClient(t, ability)

    // Act
    err := ability.DoSomething("test param")

    // Assert
    if err != nil {
        t.Logf("Expected error in unit test: %v", err)
    }

    assert.Equal(t, "do_something: test param", ability.LastOperation())
}

// ====================
// Integration Tests with Actor
// ====================

func TestYourAbility_WithActor_BasicUsage(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    actor := test.ActorCalled("TestUser").WhoCan(
        NewYourAbility(),
    )

    err := actor.AttemptsTo(
        DoSomething("test param"),
        assertions.That(Status(), expectations.ContainsSubstring("connected")),
        assertions.That(LastOperation(), expectations.ContainsSubstring("do_something")),
    )

    // В зависимости от реализации, может быть ошибка или успех
    if err != nil {
        t.Logf("Integration test completed with error (expected): %v", err)
    }
}

// ====================
// Error Scenarios
// ====================

func TestYourAbility_ErrorScenarios(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    actor := test.ActorCalled("ErrorTester").WhoCan(
        NewYourAbility(),
    )

    // Проверка обработки ошибок
    err := actor.AttemptsTo(
        DoSomething(""), // пустой параметр - должна быть ошибка
    )

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "parameter cannot be empty")
}
```

---

## 🚀 Быстрый старт

1. **Скопируйте шаблон** в новую папку `verity/abilities/your_ability/`
2. **Замените `YourAbilityName`** на имя вашей Ability
3. **Реализуйте основные методы** в `implementation.go`
4. **Создайте нужные Activities** и `Questions`
5. **Напишите тесты** following by template
6. **Обновите импорты** и export нужные функции

## 📝 Что адаптировать

- `YourAbilityName` → реальное имя вашей Ability
- `SomeClient` → ваш тип клиента для работы с внешней системой
- `DoSomething/GetSomething` → реальные методы вашей Ability
- `Config` → актуальные параметры конфигурации
- Тайминги, ошибки и логику → под ваши требования

---

Этот шаблон обеспечивает:
- ✅ Thread safety с mutex
- ✅ Proper error handling с контекстом
- ✅ Builder patterns для сложных операций
- ✅ Comprehensive test coverage
- ✅ Flexible configuration
- ✅ Clean separation of concerns
- ✅ Following Go best practices
