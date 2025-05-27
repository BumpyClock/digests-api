# ADR-001: Clean Architecture with Core Library Separation

## Status
Accepted

## Context
The original codebase mixed business logic with HTTP handling and external dependencies. This made the code difficult to test, maintain, and reuse in different contexts (e.g., CLI tools, desktop apps).

## Decision
We will implement Clean Architecture principles by:
1. Creating a `core` package containing all business logic
2. Defining interfaces for external dependencies
3. Implementing infrastructure adapters separately
4. Keeping the API layer thin, focused only on HTTP concerns

### Package Structure
```
src/
├── core/           # Business logic (no external dependencies)
│   ├── domain/     # Pure domain models
│   ├── feed/       # Feed parsing service
│   ├── search/     # Search service
│   ├── share/      # Share service
│   └── interfaces/ # Contracts for external dependencies
├── infrastructure/ # External dependency implementations
│   ├── cache/      # Cache implementations
│   ├── http/       # HTTP client implementations
│   └── logger/     # Logger implementations
├── api/           # HTTP API layer
│   ├── handlers/  # HTTP handlers
│   ├── dto/       # Request/response models
│   └── middleware/# HTTP middleware
└── cmd/          # Application entry points
```

## Consequences

### Positive
- Business logic is completely independent of frameworks
- Easy to test core logic in isolation
- Can reuse core package in different applications
- Clear separation of concerns
- Easy to swap infrastructure implementations

### Negative
- More boilerplate code for interfaces and adapters
- Additional mapping between domain models and DTOs
- Slightly more complex project structure

### Neutral
- Requires discipline to maintain architectural boundaries
- Team needs to understand dependency injection patterns