# ADR-003: Caching Strategy

## Status
Accepted

## Context
Feed parsing is an expensive operation that involves:
- Network requests to external servers
- XML/RSS parsing
- Data transformation

We needed a caching strategy that would:
- Reduce load on external feed servers
- Improve response times
- Handle both single-server and distributed deployments
- Be simple to implement and maintain

## Decision
We implement a two-tier caching strategy:

1. **Interface-based caching** in the core layer
   - Cache interface defined in core/interfaces
   - Cache keys based on feed URLs
   - 1-hour default TTL for parsed feeds
   - Service layer handles cache logic

2. **Multiple cache implementations**:
   - **Memory cache**: For single-server deployments
     - Uses sync.Map for thread safety
     - Automatic cleanup of expired entries
     - Zero configuration required
   
   - **Redis cache**: For distributed deployments
     - Shared cache across multiple servers
     - Built-in TTL support
     - Optional based on configuration

3. **Cache key strategy**:
   ```
   feed:<url_hash>              # Single feed
   search:<query_hash>          # Search results
   share:<share_id>             # Shared feed collections
   ```

## Consequences

### Positive
- Significant performance improvement for repeated requests
- Reduced load on external feed servers (good netizen behavior)
- Flexible deployment options (single server or distributed)
- Easy to add new cache implementations
- Cache warming possible for popular feeds

### Negative
- Additional complexity in service layer
- Cache invalidation challenges
- Memory usage for in-memory cache
- Redis dependency for distributed setups
- Potential for serving stale data

### Neutral
- 1-hour TTL is a tradeoff between freshness and performance
- Need monitoring for cache hit rates
- May need cache size limits for memory implementation