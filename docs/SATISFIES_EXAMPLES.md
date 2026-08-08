# Satisfies Expectation Examples

This document provides comprehensive examples of using the `Satisfies` expectation for custom validations in Verity-BDD.

## Overview

The `Satisfies` expectation allows you to create custom validation logic using functions. It's particularly useful for complex business rules, struct comparisons, and scenarios where built-in expectations aren't sufficient.

## Basic Usage

### Simple Value Validation

```go
actor.AttemptsTo(
    ensure.That(answerable.ValueOf(age), expectations.Satisfies("is positive number", func(actual int) error {
        if actual <= 0 {
            return fmt.Errorf("expected positive value, but got %d", actual)
        }
        return nil
    })),
)

actor.AttemptsTo(
    ensure.That(answerable.ValueOf(email), expectations.Satisfies("contains valid email", func(actual string) error {
        if !strings.Contains(actual, "@") {
            return fmt.Errorf("missing @ symbol in email")
        }
        if !strings.Contains(actual, ".") {
            return fmt.Errorf("missing domain in email")
        }
        return nil
    })),
)
```

### Struct Validation

```go
type User struct {
    Name string
    Age  int
}

actor.AttemptsTo(
    ensure.That(answerable.ValueOf(user), expectations.Satisfies("has valid user data", func(actual User) error {
        if actual.Name == "" {
            return fmt.Errorf("name is empty")
        }
        if actual.Age < 18 {
            return fmt.Errorf("age %d is too young (minimum 18)", actual.Age)
        }
        if actual.Age > 100 {
            return fmt.Errorf("age %d is unrealistic (maximum 100)", actual.Age)
        }
        return nil
    })),
)
```

## Advanced Usage with go-cmp

### Struct Comparison

```go
import "github.com/google/go-cmp/cmp"

type User struct {
    Name string
    Age  int
    Tags []string
}

expected := User{Name: "Alice", Age: 25, Tags: []string{"admin", "user"}}
actual := User{Name: "Alice", Age: 25, Tags: []string{"admin", "user"}}

actor.AttemptsTo(
    ensure.That(answerable.ValueOf(actual), expectations.Satisfies("matches expected user structure", func(actual User) error {
        if diff := cmp.Diff(expected, actual); diff != "" {
            return fmt.Errorf("user struct mismatch (-expected +actual):\n%s", diff)
        }
        return nil
    })),
)
```

### Comparison with Options

```go
import (
    "github.com/google/go-cmp/cmp"
    "github.com/google/go-cmp/cmp/cmpopts"
)

type TimestampedUser struct {
    ID        int
    Name      string
    CreatedAt time.Time
    UpdatedAt time.Time
}

actor.AttemptsTo(
    ensure.That(answerable.ValueOf(actual), expectations.Satisfies("matches user ignoring timestamps", func(actual TimestampedUser) error {
        if diff := cmp.Diff(expected, actual,
            cmpopts.IgnoreFields(TimestampedUser{}, "CreatedAt", "UpdatedAt"),
            cmpopts.EquateEmpty()); diff != "" {
            return fmt.Errorf("user struct mismatch (-expected +actual):\n%s", diff)
        }
        return nil
    })),
)
```

### Slice Comparison with Sorting

```go
type Item struct {
    ID   int
    Name string
}

expected := []Item{
    {ID: 2, Name: "item2"},
    {ID: 1, Name: "item1"},
}

actual := []Item{
    {ID: 1, Name: "item1"},
    {ID: 2, Name: "item2"},
}

actor.AttemptsTo(
    ensure.That(answerable.ValueOf(actual), expectations.Satisfies("matches items ignoring order", func(actual []Item) error {
        if diff := cmp.Diff(expected, actual,
            cmpopts.SortSlices(func(a, b Item) bool { return a.ID < b.ID })); diff != "" {
            return fmt.Errorf("items slice mismatch (-expected +actual):\n%s", diff)
        }
        return nil
    })),
)
```

## Complex Business Logic Validation

### Order Validation

```go
type Order struct {
    ID        string
    Amount    float64
    Currency  string
    Status    string
    CreatedAt time.Time
    Items     []OrderItem
}

type OrderItem struct {
    ProductID string
    Quantity  int
    Price     float64
}

actor.AttemptsTo(
    ensure.That(answerable.ValueOf(order), expectations.Satisfies("has valid order data", func(actual Order) error {
        // Validate ID format
        if !strings.HasPrefix(actual.ID, "ORD-") {
            return fmt.Errorf("order ID must start with ORD-, got %s", actual.ID)
        }

        // Validate amount
        if actual.Amount <= 0 {
            return fmt.Errorf("order amount must be positive, got %f", actual.Amount)
        }

        // Validate currency
        validCurrencies := []string{"USD", "EUR", "GBP"}
        currencyValid := false
        for _, currency := range validCurrencies {
            if actual.Currency == currency {
                currencyValid = true
                break
            }
        }
        if !currencyValid {
            return fmt.Errorf("invalid currency %s, valid options: %v", actual.Currency, validCurrencies)
        }

        // Validate status
        validStatuses := []string{"pending", "confirmed", "shipped", "delivered"}
        statusValid := false
        for _, status := range validStatuses {
            if actual.Status == status {
                statusValid = true
                break
            }
        }
        if !statusValid {
            return fmt.Errorf("invalid status %s, valid options: %v", actual.Status, validStatuses)
        }

        // Validate items
        if len(actual.Items) == 0 {
            return fmt.Errorf("order must have at least one item")
        }

        // Calculate total and validate
        var calculatedTotal float64
        for _, item := range actual.Items {
            if item.Quantity <= 0 {
                return fmt.Errorf("item quantity must be positive, got %d for product %s", item.Quantity, item.ProductID)
            }
            if item.Price <= 0 {
                return fmt.Errorf("item price must be positive, got %f for product %s", item.Price, item.ProductID)
            }
            calculatedTotal += float64(item.Quantity) * item.Price
        }

        // Allow small floating point differences
        if diff := actual.Amount - calculatedTotal; diff > 0.01 || diff < -0.01 {
            return fmt.Errorf("order amount %f doesn't match calculated total %f", actual.Amount, calculatedTotal)
        }

        return nil
    })),
)
```

### Configuration Validation

```go
config := map[string]interface{}{
    "database": map[string]interface{}{
        "host":     "localhost",
        "port":     5432,
        "ssl_mode": "require",
    },
    "logging": map[string]interface{}{
        "level":  "info",
        "format": "json",
    },
    "features": []interface{}{"auth", "api", "web"},
}

actor.AttemptsTo(
    ensure.That(answerable.ValueOf(config), expectations.Satisfies("has valid configuration", func(actual map[string]interface{}) error {
        // Check required sections
        requiredSections := []string{"database", "logging", "features"}
        for _, section := range requiredSections {
            if _, exists := actual[section]; !exists {
                return fmt.Errorf("missing required config section: %s", section)
            }
        }

        // Validate database config
        dbConfig, ok := actual["database"].(map[string]interface{})
        if !ok {
            return fmt.Errorf("database config must be a map")
        }

        requiredDBFields := []string{"host", "port", "ssl_mode"}
        for _, field := range requiredDBFields {
            if _, exists := dbConfig[field]; !exists {
                return fmt.Errorf("missing required database field: %s", field)
            }
        }

        // Validate port
        port, ok := dbConfig["port"].(int)
        if !ok {
            return fmt.Errorf("database port must be an integer")
        }
        if port < 1 || port > 65535 {
            return fmt.Errorf("database port must be between 1 and 65535, got %d", port)
        }

        // Validate features
        features, ok := actual["features"].([]interface{})
        if !ok {
            return fmt.Errorf("features must be an array")
        }
        if len(features) == 0 {
            return fmt.Errorf("features array cannot be empty")
        }

        return nil
    })),
)
```

## Error Messages

The description you provide to `Satisfies` appears in test failure messages, making it clear what validation failed:

```
#actor ensures that -5 (int) is positive number failed: assertion failed for '-5 (int)': expected positive value, but got -5
```

## Best Practices

1. **Use descriptive descriptions**: Make it clear what the validation is checking
2. **Provide detailed error messages**: Include the actual and expected values in error messages
3. **Keep validation focused**: Each `Satisfies` should check one logical condition
4. **Use go-cmp for struct comparisons**: Leverage go-cmp for complex struct comparisons
5. **Handle edge cases**: Consider nil values, empty collections, and boundary conditions

## Integration with Existing Expectations

`Satisfies` works seamlessly with existing expectations:

```go
// Mix built-in and custom expectations
actor.AttemptsTo(
    ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
    ensure.That(answerable.ValueOf(responseData), expectations.Satisfies("has valid response structure", func(actual ResponseData) error {
        // Custom validation logic
        return nil
    })),
)
```

## Type Safety

`Satisfies` maintains type safety with generics:

```go
// This complete expression will not compile: the question answers int,
// while the expectation accepts string.
ensure.That(
    answerable.ValueOf(1),
    expectations.Satisfies("is positive", func(actual string) error { return nil }),
)

// Correct type
ensure.That(
    answerable.ValueOf(1),
    expectations.Satisfies("is positive", func(actual int) error { return nil }),
)
```

## Running Examples

All examples in this document are available in the `examples/satisfies_demo_test.go` file. Run them with:

```bash
go test ./examples -v -run TestSatisfies
```
