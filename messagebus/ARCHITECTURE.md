# MessageBus Architecture Issue

## Problem

The socket package uses **round-robin load balancing** between providers on the same channel. This is correct for RPC but incompatible with pub/sub messaging where one message should go to ALL subscribers.

## Current Socket Behavior

```
Consumer.Send() → Server → ONE Provider (round-robin)
```

## Required for Pub/Sub

```
Publisher → Server → ALL Providers (broadcast)
```

## Solutions

### Option 1: Modify Socket Server (Breaking Change)
Add broadcast mode to socket package - routes to all providers instead of one.

**Pros:** Clean, efficient  
**Cons:** Breaks RPC semantics, major refactor

### Option 2: Broker Pattern (Recommended)
Create a dedicated broker process that receives messages and redistributes to subscribers.

```
Publisher → Broker Channel → Broker Process → Subscriber Channels → Subscribers
```

**Pros:** No socket changes, clear separation  
**Cons:** Extra hop, more complex

### Option 3: Separate Channels Per Subscriber
Each subscriber gets unique channel, publisher sends to all channels.

**Pros:** Works with current socket  
**Cons:** Doesn't scale, complex management

## Recommendation

Implement **Option 2** - broker pattern. MessageBus becomes a client library that talks to a central broker service.
