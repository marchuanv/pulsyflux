# collections - Thread-Safe Generic Collections

Thread-safe, generic data structures for Go using built-in types with proper synchronization.

## Data Structures

### Dictionary[Key, Value]
Thread-safe map with O(1) operations.

```go
dict := collections.NewDictionary[string, int]()
dict.Add("key", 42)
value, ok := dict.Get("key")
dict.Delete("key")
```

**Performance:** O(1) for all operations  
**Best for:** Any key-value storage, registries, caches

### List[T]
Thread-safe list with linear search.

```go
list := collections.NewList[string]()
list.Add("item")
if list.Has("item") { }
list.Delete("item")
```

**Performance:** O(n) for Has/Delete, O(1) for Add  
**Best for:** Small collections (< 1000 items), ordered data

### Stack[T]
Thread-safe LIFO stack.

```go
stack := collections.NewStack[int]()
stack.Push(42)
value := stack.Pop()
top := stack.Peek()
```

**Performance:** O(1) for all operations  
**Best for:** Call stacks, undo/redo, DFS algorithms

### Queue[T]
Thread-safe FIFO queue.

```go
queue := collections.NewQueue[string]()
queue.Enqueue("task")
task := queue.Dequeue()
next := queue.Peek()
```

**Performance:** O(1) amortized  
**Best for:** Task queues, BFS algorithms, buffering

## Best Practices

### ✅ DO

**Use Dictionary for lookups:**
```go
// Good: O(1) lookup
clients := collections.NewDictionary[uuid.UUID, *Client]()
client, ok := clients.Get(id)
```

**Use Stack/Queue for their intended patterns:**
```go
// Good: Natural LIFO/FIFO usage
pending := collections.NewQueue[*Request]()
pending.Enqueue(req)
```

**Check return values:**
```go
// Good: Handle missing keys
value, ok := dict.Get(key)
if !ok {
    return fmt.Errorf("key not found")
}
```

### ❌ DON'T

**Don't use List for frequent lookups on large data:**
```go
// Bad: O(n) lookup on every call
list := collections.NewList[uuid.UUID]()
if list.Has(id) { } // Slow with 10,000+ items

// Good: Use Dictionary or map instead
seen := collections.NewDictionary[uuid.UUID, struct{}]()
if seen.Has(id) { } // Fast
```

**Don't ignore Dictionary.Add errors:**
```go
// Bad: Ignoring duplicate key error
dict.Add(key, value)

// Good: Handle the error
if err := dict.Add(key, value); err != nil {
    return fmt.Errorf("duplicate key: %w", err)
}
```

**Don't use Queue for random access:**
```go
// Bad: Queue doesn't support indexing
queue.Dequeue() // Just to peek at second item

// Good: Use List or slice if you need indexing
```

## Performance Guide

| Operation | Dictionary | List | Stack | Queue |
|-----------|-----------|------|-------|-------|
| Add/Push/Enqueue | O(1) | O(1) | O(1) | O(1) |
| Get/Has | O(1) | O(n) | - | - |
| Delete | O(1) | O(n) | - | - |
| Pop/Dequeue | - | - | O(1) | O(1) |
| Peek | - | - | O(1) | O(1) |

## Concurrency

All types are thread-safe:
- **Dictionary/List:** Use `sync.RWMutex` (multiple readers, single writer)
- **Stack/Queue:** Use `sync.Mutex` (exclusive access)

```go
// Safe: Concurrent access from multiple goroutines
dict := collections.NewDictionary[int, string]()
go func() { dict.Add(1, "a") }()
go func() { dict.Get(1) }()
```

## When to Use Alternatives

**Instead of List for large lookups:**
```go
// Use map for O(1) lookups
seen := make(map[string]struct{})
if _, ok := seen[key]; ok { }
```

**Instead of Queue for high throughput:**
```go
// Use channels for producer-consumer
tasks := make(chan Task, 100)
tasks <- task
task := <-tasks
```

**Instead of Dictionary for simple cases:**
```go
// Use sync.Map for dynamic key types
var cache sync.Map
cache.Store(key, value)
value, ok := cache.Load(key)
```

## Examples

### Client Registry
```go
type Server struct {
    clients *collections.Dictionary[uuid.UUID, *Client]
}

func (s *Server) Register(client *Client) error {
    return s.clients.Add(client.ID, client)
}

func (s *Server) GetClient(id uuid.UUID) (*Client, bool) {
    return s.clients.Get(id)
}
```

### Request Queue
```go
type Handler struct {
    pending *collections.Queue[*Request]
}

func (h *Handler) Enqueue(req *Request) {
    h.pending.Enqueue(req)
}

func (h *Handler) Process() {
    for h.pending.Len() > 0 {
        req := h.pending.Dequeue()
        req.Handle()
    }
}
```

### Undo Stack
```go
type Editor struct {
    history *collections.Stack[Command]
}

func (e *Editor) Execute(cmd Command) {
    cmd.Do()
    e.history.Push(cmd)
}

func (e *Editor) Undo() {
    if e.history.Len() > 0 {
        cmd := e.history.Pop()
        cmd.Undo()
    }
}
```
