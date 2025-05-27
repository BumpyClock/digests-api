# ADR-002: Huma Framework for API Layer

## Status
Accepted

## Context
We needed to choose a web framework for the API layer that would provide:
- Automatic OpenAPI documentation
- Request/response validation
- Clean handler interface
- Good performance
- Active maintenance

Options considered:
1. Gin - Fast but requires manual OpenAPI generation
2. Echo - Similar to Gin, manual documentation
3. Fiber - Fast but different API from standard library
4. Huma - Built on Chi, automatic OpenAPI, validation
5. Buffalo - Full framework, too heavyweight

## Decision
We chose Huma v2 framework because:
- Automatic OpenAPI 3.0 generation from Go structs
- Built-in request/response validation using struct tags
- Clean separation between business logic and HTTP concerns
- Built on Chi router (standard library compatible)
- Active development and good documentation
- Type-safe handlers with compile-time checks

## Consequences

### Positive
- No manual OpenAPI maintenance
- Automatic request validation reduces boilerplate
- Type-safe handlers prevent common errors
- Swagger UI included out of the box
- Compatible with standard HTTP middleware

### Negative
- Less mature than Gin/Echo
- Smaller community
- Specific handler signature required
- Learning curve for team unfamiliar with Huma

### Neutral
- Locked into Huma's way of doing things
- Need to follow Huma conventions for best results