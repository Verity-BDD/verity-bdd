# Создание собственных Abilities

Эта инструкция покажет, как создавать собственные Abilities для расширения возможностей Verity-BDD под ваши специфические потребности тестирования.

## 🎯 Что такое Ability?

**Ability** (способность) - это возможность, которую Actor может использовать для взаимодействия с системой. В паттерне Screenplay, Abilities определяют **ЧТО** Actor может делать, а не **КАК** он это делает.

### Примеры существующих Abilities:
- `CallAnAPI` - выполнение HTTP-запросов
- `ConnectToDatabase` - работа с базами данных
- `UseFileSystem` - операции с файловой системой

## 🏗️ Архитектура Ability

### Базовый интерфейс

```go
// verity/abilities/ability.go
package abilities

// Ability - маркерный интерфейс для всех способностей
type Ability interface{}
```

### Поиск Ability в Actor

```go
// Actor находит Ability по типу
ability, err := actor.AbilityTo(&targetAbilityType{})
if err != nil {
    return fmt.Errorf("actor does not have required ability: %w", err)
}

// Приведение к конкретному типу
specificAbility := ability.(SpecificAbility)
```

## 📋 Пошаговая инструкция создания Ability

### Шаг 1: Определите интерфейс Ability

Создайте интерфейс, который:
1. Наследует `abilities.Ability`
2. Определяет методы, специфичные для вашей Ability

```go
package custom

import "github.com/verity-bdd/verity-bdd/verity_abilities"

// FileManagerAbility - способность для работы с файлами
type FileManagerAbility interface {
    abilities.Ability

    // Core operations
    ReadFile(path string) (string, error)
    WriteFile(path string, content string) error
    DeleteFile(path string) error

    // State management
    LastOperation() string
    SetWorkingDirectory(dir string) error
}
```

### Шаг 2: Создайте приватную реализацию

```go
// fileManagerAbility - приватная реализация
type fileManagerAbility struct {
    workingDir   string
    lastOperation string
}

// Конструктор
func ManageFiles() FileManagerAbility {
    return &fileManagerAbility{
        workingDir: ".",
        lastOperation: "none",
    }
}

// Методы реализации
func (f *fileManagerAbility) ReadFile(path string) (string, error) {
    fullPath := filepath.Join(f.workingDir, path)
    content, err := os.ReadFile(fullPath)
    if err != nil {
        f.lastOperation = fmt.Sprintf("read error: %s", path)
        return "", fmt.Errorf("failed to read file %s: %w", path, err)
    }

    f.lastOperation = fmt.Sprintf("read: %s", path)
    return string(content), nil
}

func (f *fileManagerAbility) WriteFile(path string, content string) error {
    fullPath := filepath.Join(f.workingDir, path)
    err := os.WriteFile(fullPath, []byte(content), 0644)
    if err != nil {
        f.lastOperation = fmt.Sprintf("write error: %s", path)
        return fmt.Errorf("failed to write file %s: %w", path, err)
    }

    f.lastOperation = fmt.Sprintf("write: %s", path)
    return nil
}

func (f *fileManagerAbility) DeleteFile(path string) error {
    fullPath := filepath.Join(f.workingDir, path)
    err := os.Remove(fullPath)
    if err != nil {
        f.lastOperation = fmt.Sprintf("delete error: %s", path)
        return fmt.Errorf("failed to delete file %s: %w", path, err)
    }

    f.lastOperation = fmt.Sprintf("delete: %s", path)
    return nil
}

func (f *fileManagerAbility) LastOperation() string {
    return f.lastOperation
}

func (f *fileManagerAbility) SetWorkingDirectory(dir string) error {
    if !filepath.IsAbs(dir) {
        abs, err := filepath.Abs(dir)
        if err != nil {
            return fmt.Errorf("failed to get absolute path: %w", err)
        }
        dir = abs
    }

    if _, err := os.Stat(dir); os.IsNotExist(err) {
        return fmt.Errorf("directory does not exist: %s", dir)
    }

    f.workingDir = dir
    return nil
}
```

### Шаг 3: Создайте фабричные методы (опционально)

Для удобства использования создайте именованные конструкторы:

```go
// Разные способы создания Ability
func ManageFiles() FileManagerAbility {
    return &fileManagerAbility{workingDir: "."}
}

func ManageFilesIn(directory string) FileManagerAbility {
    return &fileManagerAbility{workingDir: directory}
}

func ManageFilesWithConfig(config FileManagerConfig) FileManagerAbility {
    return &fileManagerAbility{
        workingDir: config.WorkingDirectory,
        // другие параметры конфигурации
    }
}

// Конфигурация для сложных Ability
type FileManagerConfig struct {
    WorkingDirectory string
    CreateDirs        bool
    BackupOnDelete    bool
}
```

### Шаг 4: Интегрируйте с Activities

Создайте Activities, которые используют вашу Ability:

```go
package custom

import (
    "github.com/verity-bdd/verity-bdd"
)

// ReadFileActivity - чтение файла
type ReadFileActivity struct {
    path string
}

func ReadFile(path string) *ReadFileActivity {
    return &ReadFileActivity{path: path}
}

func (r *ReadFileActivity) PerformAs(actor core.Actor) error {
    // Получаем Ability от Actor
    ability, err := actor.AbilityTo(&fileManagerAbility{})
    if err != nil {
        return fmt.Errorf("actor does not have file management ability: %w", err)
    }

    fileManager := ability.(FileManagerAbility)
    _, err = fileManager.ReadFile(r.path)
    return err
}

func (r *ReadFileActivity) Description() string {
    return fmt.Sprintf("reads file: %s", r.path)
}
```

### Шаг 5: Создайте Questions для проверки состояния

```go
// FileContentQuestion - вопрос о содержимом файла
type FileContentQuestion struct {
    path string
}

func FileContent(path string) *FileContentQuestion {
    return &FileContentQuestion{path: path}
}

func (f *FileContentQuestion) AnsweredBy(actor core.Actor) (string, error) {
    ability, err := actor.AbilityTo(&fileManagerAbility{})
    if err != nil {
        return "", fmt.Errorf("actor does not have file management ability: %w", err)
    }

    fileManager := ability.(FileManagerAbility)
    return fileManager.ReadFile(f.path)
}

func (f *FileContentQuestion) Description() string {
    return fmt.Sprintf("content of file: %s", f.path)
}
```

## 🔧 Использование вашей Ability

### Базовое использование

```go
func TestFileOperations(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    // Создаем Actor с нашей новой Ability
    actor := test.ActorCalled("FileUser").WhoCan(
        custom.ManageFilesIn("/tmp/test"),
    )

    // Используем Activities
    err := actor.AttemptsTo(
        custom.WriteFile("test.txt", "Hello, World!"),
        ensure.That(custom.FileContent("test.txt"), expectations.ContainsSubstring("Hello")),
    )

    require.NoError(t, err)
}
```

### Композиция с другими Abilities

```go
func TestAPIAndFileOperations(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    actor := test.ActorCalled("IntegrationTester").WhoCan(
        api.CallAnApiAt("https://api.example.com"),
        custom.ManageFilesIn("./test-data"),
    )

    err := actor.AttemptsTo(
        // Сначала получаем данные из API
        api.SendGetRequest("/users/1"),
        // Затем сохраняем их в файл
        custom.WriteFile("user.json", api.LastResponseBody{}),
        // Проверяем содержимое файла
        ensure.That(custom.FileContent("user.json"), expectations.ContainsSubstring("name")),
    )

    require.NoError(t, err)
}
```

## ⚡ Advanced Patterns

### Pattern 1: Builder для сложных Activities

```go
type WriteFileActivity struct {
    path    string
    content string
    mode    os.FileMode
    backup  bool
}

func WriteFile(path string) *WriteFileActivity {
    return &WriteFileActivity{
        path:    path,
        mode:    0644,
        backup:  false,
    }
}

func (w *WriteFileActivity) WithContent(content string) *WriteFileActivity {
    w.content = content
    return w
}

func (w *WriteFileActivity) WithMode(mode os.FileMode) *WriteFileActivity {
    w.mode = mode
    return w
}

func (w *WriteFileActivity) WithBackup() *WriteFileActivity {
    w.backup = true
    return w
}

// Использование:
err := actor.AttemptsTo(
    custom.WriteFile("config.json").
        WithContent(configData).
        WithMode(0600).
        WithBackup(),
)
```

### Pattern 2: State Management между вызовами

```go
type DatabaseConnectionAbility interface {
    abilities.Ability
    Connect(dsn string) error
    Disconnect() error
    Execute(query string, args ...interface{}) (*sql.Rows, error)
    LastQuery() string
    LastError() error
}

type databaseConnectionAbility struct {
    db         *sql.DB
    lastQuery  string
    lastError  error
    isConnected bool
    mutex      sync.RWMutex
}

func (d *databaseConnectionAbility) Execute(query string, args ...interface{}) (*sql.Rows, error) {
    d.mutex.Lock()
    defer d.mutex.Unlock()

    d.lastQuery = query

    if !d.isConnected {
        d.lastError = fmt.Errorf("not connected to database")
        return nil, d.lastError
    }

    rows, err := d.db.Query(query, args...)
    d.lastError = err
    return rows, err
}
```

### Pattern 3: Error Handling и Retry Logic

```go
type ResilientAPIAbility interface {
    abilities.Ability
    SendRequest(req *http.Request) (*http.Response, error)
    WithRetryPolicy(policy RetryPolicy) ResilientAPIAbility
    WithTimeout(timeout time.Duration) ResilientAPIAbility
}

type resilientAPIAbility struct {
    client      *http.Client
    retryPolicy RetryPolicy
    timeout     time.Duration
}

func (r *resilientAPIAbility) SendRequest(req *http.Request) (*http.Response, error) {
    var lastErr error

    for attempt := 0; attempt <= r.retryPolicy.MaxRetries; attempt++ {
        if attempt > 0 {
            time.Sleep(r.retryPolicy.Delay(attempt))
        }

        resp, err := r.client.Do(req)
        if err == nil {
            return resp, nil
        }

        lastErr = err

        if !r.retryPolicy.ShouldRetry(err, attempt) {
            break
        }
    }

    return nil, fmt.Errorf("request failed after %d attempts: %w", r.retryPolicy.MaxRetries+1, lastErr)
}
```

## 🧪 Тестирование вашей Ability

### Unit тесты для реализации

```go
func TestFileManagerAbility_ReadFile(t *testing.T) {
    // Arrange
    tempDir := t.TempDir()
    ability := custom.ManageFilesIn(tempDir)
    testFile := filepath.Join(tempDir, "test.txt")

    // Создаем тестовый файл
    require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

    // Act
    content, err := ability.ReadFile("test.txt")

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "test content", content)
    assert.Equal(t, "read: test.txt", ability.LastOperation())
}

func TestFileManagerAbility_ReadFile_NotFound(t *testing.T) {
    // Arrange
    ability := custom.ManageFilesIn(t.TempDir())

    // Act
    _, err := ability.ReadFile("nonexistent.txt")

    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "failed to read file")
    assert.Equal(t, "read error: nonexistent.txt", ability.LastOperation())
}
```

### Интеграционные тесты с Actor

```go
func TestFileManagerIntegration(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    actor := test.ActorCalled("FileTester").WhoCan(
        custom.ManageFilesIn(t.TempDir()),
    )

    err := actor.AttemptsTo(
        custom.WriteFile("integration.txt", "integration test"),
        ensure.That(custom.FileContent("integration.txt"), expectations.Equals("integration test")),
    )

    require.NoError(t, err)
}
```

## 📋 Best Practices

### ✅ Do's
1. **Используйте интерфейсы** - отделяйте контракты от реализации
2. **Создавайте именованные конструкторы** - для разных сценариев использования
3. **Храните состояние** только если это необходимо между вызовами
4. **Оборачивайте ошибки** с контекстом
5. **Используйте RWMutex** для защиты состояния в concurrent scenarios
6. **Пишите тесты** для каждого метода Ability
7. **Следуйте Go naming conventions**

### ❌ Don'ts
1. **Не храните** в Ability большую mutable state
2. **Не создавайте** глобальные переменные в Ability
3. **Не игнорируйте** ошибки - всегда возвращайте их
4. **Не смешивайте** concerns - одна Ability = одна ответственность
5. **Не забывайте** про thread safety

## 🎯 Примеры готовых Ability

### Database Ability
```go
type DatabaseAbility interface {
    abilities.Ability
    Query(query string, args ...interface{}) (*sql.Rows, error)
    Execute(query string, args ...interface{}) (sql.Result, error)
    BeginTx() (*sql.Tx, error)
}

func ConnectToPostgreSQL(dsn string) DatabaseAbility {
    // implementation
}
```

### Redis Ability
```go
type RedisAbility interface {
    abilities.Ability
    Set(key string, value interface{}) error
    Get(key string) (string, error)
    Del(key string) error
}

func ConnectToRedis(addr string) RedisAbility {
    // implementation
}
```

### WebSocket Ability
```go
type WebSocketAbility interface {
    abilities.Ability
    Connect(url string) error
    Send(message []byte) error
    Receive(timeout time.Duration) ([]byte, error)
    Close() error
}
```

---

## 🚀 Следующие шаги

1. **Изучите примеры** в [docs/examples/](examples/)
2. **Используйте шаблоны** из [docs/templates/](templates/)
3. **Посмотрите существующие Abilities** в исходном коде проекта
4. **Изучите готовый пример** FileSystemAbility с тестами в [examples/ability/](../examples/ability/)

Удачи в создании мощных и гибких тестов с помощью Verity-BDD! 🎉

