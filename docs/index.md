# Verity-BDD Documentation

Welcome to the comprehensive documentation for Verity-BDD - a Screenplay Pattern testing framework for Go.

## 📚 Documentation Structure

### 🚀 Getting Started
- [**Main README**](../README.md) - Quick start guide and overview
- [**Core Concepts**](../README.md#core-api) - Understanding Screenplay Pattern basics

### 🛠️ Developer Guides
- [**Creating Custom Abilities**](abilities.md) - Complete guide to creating your own abilities
- [**Examples & Patterns**](examples/) - Real-world examples and best practices
- [**Templates**](templates/) - Ready-to-use templates for common tasks

### 🔧 Advanced Topics
- [**API Testing**](../README.md#http-api-testing) - HTTP API testing capabilities
- [**Task Composition**](../README.md#task-composition) - Building complex workflows
- [**Multiple Actors**](../README.md#multiple-actors) - Working with different test personas

## 🎯 Quick Links

### For Beginners
1. **Read the [Main README](../README.md)** - Understand basic concepts
2. **Try the [Examples](../examples/)** - See working code
3. **Create your first [Custom Ability](abilities.md)** - Extend the framework

### For Advanced Users
1. **[Custom Ability Guide](abilities.md)** - Create reusable components
2. **[Example Implementations](examples/)** - Learn from real code
3. **[Development Templates](templates/)** - Speed up development

## 🏗️ Core Concepts

### Actors
Actors represent users or systems that interact with your application:
```go
test := verity.NewVerityTest(t, verity.Scene{})

actor := test.ActorCalled("APITester").WhoCan(api.CallAnApiAt("https://api.example.com"))
```

### Abilities
Abilities enable actors to interact with different parts of your system:
```go
apiAbility := api.CallAnApiAt("https://api.example.com")
```

### Activities
Activities represent actions that actors perform:
```go
actor.AttemptsTo(
    api.SendGetRequest("/users"),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
)
```

`AttemptsTo` reports failures through the test context and returns no value.

## 🚀 Quick Start

### 1. Install
```bash
go get github.com/verity-bdd/verity-bdd
```

### 2. Basic Test
```go
func TestAPI(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    actor := test.ActorCalled("APITester").WhoCan(
        api.CallAnApiAt("https://jsonplaceholder.typicode.com"),
    )

    actor.AttemptsTo(
        api.SendGetRequest("/posts"),
        ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
    )
}
```

### 3. Create Custom Ability
Ready to extend the framework? Check out our [Creating Custom Abilities guide](abilities.md).

## 📖 Documentation Sections

### [Creating Custom Abilities](abilities.md)
Learn how to create your own abilities for:
- Database testing
- File system operations
- Third-party service integrations
- Custom protocols and APIs

### [Examples](examples/)
Real-world examples including:
- Complete ability implementations
- Integration patterns
- Testing strategies
- Common use cases

### [Templates](templates/)
Ready-to-use templates for:
- New ability structure
- Test patterns
- Common integrations

## 🤝 Contributing

Have an idea to improve the documentation? Found something unclear?

- Check out the [main repository](https://github.com/verity-bdd/verity-bdd)
- Open an issue for documentation improvements
- Submit pull requests for new examples and guides

---

**Next Steps**: Start with [Creating Custom Abilities](abilities.md) to extend Verity-BDD for your testing needs.
