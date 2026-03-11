# Implementation Plan: Message Broker Delivery Guarantees

## Overview

This implementation plan extends the existing PulsyFlux broker with reliable message delivery, deterministic ordering, state synchronization, and reconnection recovery. The implementation follows a phased approach, building core components first, then integrating them into the broker, and finally adding advanced features like snapshots and reconnection management.

The implementation uses Go and builds on the existing broker and tcp-conn packages. All components are designed for 2-10 concurrent clients with 10-30 messages/second per client, targeting <50ms p99 latency and 300 messages/second total throughput.

## Tasks

- [ ] 1. Set up core data structures and interfaces
  - Create `pulsyflux/broker/delivery/` directory for delivery guarantee components
  - Define Message, MessageEnvelope, and protocol message types
  - Define SessionConfig with validation
  - Define Session struct with component references
  - Define Client struct with reconnection fields
  - _Requirements: 1.1, 2.1, 3.1, 6.1, 8.1_

- [ ]* 1.1 Write unit tests for configuration validation
  - Test valid configuration values (tick interval, ack timeout, reconnect timeout)
  - Test invalid configuration values (out of range)
  - Test edge cases (boundary values)
  - _Requirements: 6.6, 8.4_

- [ ] 2. Implement Sequence Manager
  - [ ] 2.1 Create SequenceManager struct with atomic operations
    - Implement Next() for monotonic sequence generation
    - Implement Current() for reading current sequence
    - Implement Reset() for recovery scenarios
    - _Requirements: 3.1_

  - [ ]* 2.2 Write property test for monotonic sequence numbers
    - **Property 8: Monotonic Sequence Numbers**
    - **Validates: Requirements 3.1**

  - [ ]* 2.3 Write unit tests for SequenceManager
    - Test first sequence assignment (should be 1)
    - Test subsequent sequences (should increment)
    - Test concurrent access from multiple goroutines
    - Test Reset() functionality
    - _Requirements: 3.1_

- [ ] 3. Implement Message Store (in-memory with persistence interface)
  - [ ] 3.1 Create MessageStore interface
    - Define Store(), Get(), Delete() methods
    - Define GetUnacknowledged(), GetBySequenceRange() methods
    - Define UpdateDeliveryAttempt(), MarkAcknowledged() methods
    - Define GetHistorical(), Close() methods
    - _Requirements: 1.2, 1.3, 11.1, 11.6_

  - [ ] 3.2 Implement InMemoryMessageStore
    - Use map with mutex for thread-safe storage
    - Implement atomic write operations
    - Track acknowledged status and delivery attempts
    - Support sequence range queries
    - _Requirements: 1.2, 11.4_

  - [ ]* 3.3 Write property test for message storage round trip
    - **Property 7: Message Storage Round Trip**
    - **Validates: Requirements 2.6**

  - [ ]* 3.4 Write unit tests for MessageStore
    - Test Store() and Get() operations
    - Test Delete() removes message
    - Test GetUnacknowledged() returns only unacked messages
    - Test GetBySequenceRange() with various ranges
    - Test concurrent access from multiple goroutines
    - Test MarkAcknowledged() updates status
    - _Requirements: 1.2, 1.3, 11.1_

- [ ] 4. Implement Idempotency Manager
  - [ ] 4.1 Create IdempotencyManager interface and implementation
    - Implement CheckAndStore() for atomic duplicate detection
    - Implement Remove() for cleanup
    - Implement Cleanup() with retention policy (60 seconds)
    - Use map with mutex for thread-safe access
    - Track creation timestamps for expiration
    - _Requirements: 2.1, 2.2, 2.3_

  - [ ]* 4.2 Write property test for duplicate message deduplication
    - **Property 5: Duplicate Message Deduplication**
    - **Validates: Requirements 2.2**

  - [ ]* 4.3 Write property test for idempotency key retention
    - **Property 6: Idempotency Key Retention**
    - **Validates: Requirements 2.3**

  - [ ]* 4.4 Write unit tests for IdempotencyManager
    - Test CheckAndStore() returns true for new key
    - Test CheckAndStore() returns false for duplicate key
    - Test Cleanup() removes expired keys
    - Test Cleanup() retains recent keys
    - Test concurrent access from multiple goroutines
    - _Requirements: 2.1, 2.2, 2.3_

- [ ] 5. Implement Acknowledgment Manager
  - [ ] 5.1 Create AckManager interface and implementation
    - Implement TrackDelivery() to register pending acks
    - Implement Acknowledge() to mark messages as delivered
    - Implement GetTimedOut() to find expired acks
    - Use timer goroutines for timeout detection
    - Background worker to check timeouts periodically
    - _Requirements: 1.3, 1.4, 1.5_

  - [ ]* 5.2 Write property test for acknowledgment lifecycle
    - **Property 3: Acknowledgment Lifecycle**
    - **Validates: Requirements 1.3, 1.4, 1.5**

  - [ ]* 5.3 Write unit tests for AckManager
    - Test TrackDelivery() registers message
    - Test Acknowledge() removes from pending
    - Test GetTimedOut() returns expired messages
    - Test timeout triggers after configured duration (500ms)
    - Test concurrent acknowledgments
    - Test Stop() cleans up goroutines
    - _Requirements: 1.3, 1.4, 1.5_

- [ ] 6. Checkpoint - Core components complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. Implement Message Validator with rate limiting
  - [ ] 7.1 Create MessageValidator interface
    - Define Validate() method for message validation
    - Define CheckRateLimit() method for rate limiting
    - _Requirements: 7.1, 7.2, 7.6_

  - [ ] 7.2 Implement RateLimiter
    - Track message counts per client per second
    - Use sliding window for rate calculation
    - Reset window after 1 second
    - Reject messages exceeding 100 msg/s limit
    - _Requirements: 7.6_

  - [ ]* 7.3 Write property test for rate limiting
    - **Property 30: Rate Limiting**
    - **Validates: Requirements 7.6**

  - [ ]* 7.4 Write unit tests for MessageValidator
    - Test rate limit under threshold (should accept)
    - Test rate limit at threshold (should accept)
    - Test rate limit over threshold (should reject)
    - Test window reset after 1 second
    - Test multiple clients with independent limits
    - _Requirements: 7.6_

- [ ] 8. Implement Tick Engine for deterministic ordering
  - [ ] 8.1 Create TickEngine struct
    - Implement Start() to begin tick processing
    - Implement processTick() to collect and order messages
    - Implement collectMessages() to gather from all clients
    - Use time.Ticker for tick intervals
    - Buffer messages per client between ticks
    - _Requirements: 8.1, 8.4, 8.5_

  - [ ] 8.2 Implement deterministic ordering logic
    - Sort messages by sequence number (primary)
    - Sort by client ID for tie-breaking (secondary)
    - Send empty placeholders for clients with no messages
    - Broadcast messages in deterministic order
    - _Requirements: 8.1, 8.2, 8.3, 8.7_

  - [ ]* 8.3 Write property test for deterministic message ordering
    - **Property 32: Deterministic Message Ordering**
    - **Validates: Requirements 8.1, 8.2, 8.3**

  - [ ]* 8.4 Write unit tests for TickEngine
    - Test tick processing with messages from all clients
    - Test tick processing with messages from some clients
    - Test tick processing with no messages
    - Test empty placeholder generation
    - Test deterministic ordering (same input = same output)
    - Test tick interval timing
    - _Requirements: 8.1, 8.2, 8.3, 8.5, 8.7_

- [ ] 9. Implement Snapshot Manager with delta encoding
  - [ ] 9.1 Create SnapshotManager struct
    - Implement CreateSnapshot() to generate state snapshots
    - Implement GetLatest() to retrieve most recent snapshot
    - Maintain retention of 3 most recent snapshots
    - Track snapshot sequence numbers
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [ ] 9.2 Implement delta encoding
    - Implement GenerateDelta() to create delta updates
    - Compare delta size to snapshot size (80% threshold)
    - Fall back to full snapshot if delta too large
    - Include base sequence number in deltas
    - _Requirements: 5.1, 5.2, 5.5_

  - [ ]* 9.3 Write property test for delta application correctness
    - **Property 20: Delta Application Correctness**
    - **Validates: Requirements 5.3, 5.6**

  - [ ]* 9.4 Write property test for snapshot retention policy
    - **Property 16: Snapshot Retention Policy**
    - **Validates: Requirements 4.4**

  - [ ]* 9.5 Write unit tests for SnapshotManager
    - Test CreateSnapshot() generates valid snapshot
    - Test snapshot retention (keeps only 3 most recent)
    - Test GetLatest() returns most recent snapshot
    - Test snapshot includes sequence number
    - Test GenerateDelta() creates delta with changes only
    - Test delta size threshold (>80% triggers full snapshot)
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 5.1, 5.5_

- [ ] 10. Implement Reconnection Manager with token-based recovery
  - [ ] 10.1 Create ReconnectionManager struct
    - Implement IssueToken() to generate reconnection tokens
    - Implement ValidateToken() to check token validity
    - Implement HandleReconnect() to restore client session
    - Track token expiration (configurable 30-600s)
    - Clean up expired tokens periodically
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [ ] 10.2 Implement reconnection recovery flow
    - Send latest snapshot to reconnecting client
    - Send all messages since snapshot sequence number
    - Send buffered messages from disconnection period
    - Restore client connection and state
    - _Requirements: 6.4, 6.7_

  - [ ]* 10.3 Write property test for token-based reconnection
    - **Property 24: Token-Based Reconnection**
    - **Validates: Requirements 6.3**

  - [ ]* 10.4 Write property test for reconnection recovery sequence
    - **Property 25: Reconnection Recovery Sequence**
    - **Validates: Requirements 6.4**

  - [ ]* 10.5 Write unit tests for ReconnectionManager
    - Test IssueToken() generates unique tokens
    - Test ValidateToken() accepts valid tokens
    - Test ValidateToken() rejects invalid tokens
    - Test ValidateToken() rejects expired tokens
    - Test HandleReconnect() sends snapshot + messages
    - Test CleanupExpiredTokens() removes expired tokens
    - Test message buffering during disconnection (max 1000)
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.7_

- [ ] 11. Checkpoint - Advanced components complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 12. Implement Reorder Buffer (client-side)
  - [ ] 12.1 Create ReorderBuffer struct
    - Implement Add() to buffer out-of-order messages
    - Deliver messages when sequence gaps are filled
    - Track expected sequence number
    - Hold messages for 2 seconds before timeout
    - _Requirements: 3.4, 3.5, 3.6_

  - [ ] 12.2 Implement timeout and retransmission logic
    - Implement CheckTimeouts() to find expired gaps
    - Request retransmission for persistent gaps (>2s)
    - _Requirements: 3.6, 3.7_

  - [ ]* 12.3 Write property test for reorder buffer behavior
    - **Property 11: Reorder Buffer Behavior**
    - **Validates: Requirements 3.4, 3.5, 3.6**

  - [ ]* 12.4 Write unit tests for ReorderBuffer
    - Test Add() with expected sequence (immediate delivery)
    - Test Add() with future sequence (buffer)
    - Test Add() with past sequence (discard duplicate)
    - Test gap filling delivers consecutive messages
    - Test CheckTimeouts() identifies expired gaps
    - Test timeout threshold (2 seconds)
    - _Requirements: 3.4, 3.5, 3.6, 3.7_

- [ ] 13. Implement Session Metrics
  - [ ] 13.1 Create SessionMetrics struct
    - Track messages processed, delivered, rejected
    - Track duplicates detected
    - Track reconnection events
    - Track snapshots created and deltas generated
    - Track per-client message counts
    - Track latency samples per client
    - Track rejection reasons
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6_

  - [ ] 13.2 Implement LatencyTracker
    - Record latency samples (max 1000 samples)
    - Calculate P99 latency
    - _Requirements: 12.2_

  - [ ]* 13.3 Write property tests for metrics accuracy
    - **Property 45: Message Rate Metrics**
    - **Property 46: Latency Metrics**
    - **Property 47: Duplicate Detection Metrics**
    - **Validates: Requirements 12.1, 12.2, 12.3**

  - [ ]* 13.4 Write unit tests for SessionMetrics
    - Test counter increments (messages, duplicates, rejections)
    - Test per-client metrics tracking
    - Test LatencyTracker records samples
    - Test LatencyTracker calculates P99 correctly
    - Test rejection reason categorization
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6_

- [ ] 14. Integrate components into Session
  - [ ] 14.1 Create Session struct with all components
    - Initialize SequenceManagers (per-client)
    - Initialize MessageStore
    - Initialize AckManager
    - Initialize IdempotencyManager
    - Initialize SnapshotManager
    - Initialize TickEngine
    - Initialize MessageValidator
    - Initialize ReconnectionManager
    - Initialize SessionMetrics
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1, 6.1, 7.1, 8.1_

  - [ ] 14.2 Implement message receive flow
    - Validate message format
    - Check rate limit
    - Assign sequence number
    - Check idempotency (deduplicate)
    - Persist to message store
    - Buffer in tick engine
    - _Requirements: 1.1, 1.2, 2.1, 2.2, 3.1, 7.6_

  - [ ] 14.3 Implement message broadcast flow
    - Tick engine collects and orders messages
    - Broadcast to all clients in deterministic order
    - Track delivery with AckManager
    - Update metrics
    - _Requirements: 1.1, 8.1, 8.2, 8.3_

  - [ ] 14.4 Implement acknowledgment handling
    - Receive ack from client
    - Mark message as acknowledged in store
    - Remove from AckManager
    - Delete from message store within 100ms
    - Update metrics
    - _Requirements: 1.3, 1.5_

  - [ ]* 14.5 Write integration tests for message flow
    - Test end-to-end: receive → validate → sequence → persist → broadcast → ack
    - Test multiple clients sending messages concurrently
    - Test message ordering across clients
    - Test duplicate detection in full flow
    - Test rate limiting in full flow
    - _Requirements: 1.1, 1.2, 1.3, 2.2, 3.1, 7.6, 8.1_

- [ ] 15. Implement client join and reconnection flows
  - [ ] 15.1 Implement client join flow
    - Issue reconnection token
    - Send latest snapshot
    - Add client to session
    - Initialize sequence manager for client
    - _Requirements: 4.1, 6.1_

  - [ ] 15.2 Implement client disconnect handling
    - Mark client as disconnected
    - Start retention timer (configurable 30-600s)
    - Buffer subsequent messages (max 1000)
    - _Requirements: 6.2, 6.7_

  - [ ] 15.3 Implement client reconnect flow
    - Validate reconnection token
    - Send latest snapshot
    - Send messages since snapshot
    - Send buffered messages
    - Resume normal operation
    - _Requirements: 6.3, 6.4, 6.7_

  - [ ]* 15.4 Write integration tests for reconnection flow
    - Test disconnect → buffer → reconnect → recover
    - Test token expiration removes client
    - Test buffer overflow (>1000 messages)
    - Test snapshot + catch-up delivery
    - _Requirements: 6.2, 6.3, 6.4, 6.5, 6.7_

- [ ] 16. Checkpoint - Session integration complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 17. Implement protocol extensions
  - [ ] 17.1 Define protocol message types
    - MessageEnvelope (with sequence number, idempotency key)
    - AckMessage (message acknowledgment)
    - SnapshotMessage (state snapshot)
    - DeltaMessage (delta update)
    - ErrorMessage (error responses)
    - ControlMessage (join, reconnect)
    - _Requirements: 1.1, 2.1, 3.3, 4.1, 5.1, 6.1_

  - [ ] 17.2 Implement message encoding/decoding
    - JSON serialization for all message types
    - Include metadata (sequence numbers, timestamps)
    - Handle backward compatibility
    - _Requirements: 1.1, 3.3_

  - [ ]* 17.3 Write unit tests for protocol messages
    - Test encoding/decoding for all message types
    - Test JSON serialization round-trip
    - Test required fields validation
    - _Requirements: 1.1, 2.1, 3.3_

- [ ] 18. Implement error handling
  - [ ] 18.1 Implement message validation errors
    - Reject invalid message format
    - Send error response to client
    - Log error with details
    - Increment rejection metrics
    - _Requirements: 7.3, 7.7_

  - [ ] 18.2 Implement rate limit errors
    - Reject excess messages
    - Send rate limit error response
    - Log warning with client ID
    - Increment rejection metrics
    - _Requirements: 7.6, 7.7, 12.7_

  - [ ] 18.3 Implement storage errors
    - Handle message store unavailable
    - Handle write failures (retry with backoff)
    - Handle read failures (skip and continue)
    - Log errors and update metrics
    - _Requirements: 11.1_

  - [ ] 18.4 Implement network errors
    - Handle client connection failures
    - Handle send failures (retry once)
    - Trigger redelivery via ack timeout
    - Log failures and update metrics
    - _Requirements: 1.4_

  - [ ] 18.5 Implement acknowledgment errors
    - Handle ack timeout (redeliver)
    - Handle invalid ack (ignore)
    - Handle duplicate ack (ignore)
    - Move to dead letter after max attempts (5)
    - _Requirements: 1.4_

  - [ ] 18.6 Implement reconnection errors
    - Handle invalid token (reject)
    - Handle expired token (reject)
    - Handle buffer overflow (drop oldest)
    - _Requirements: 6.3, 6.5, 6.7_

  - [ ]* 18.7 Write unit tests for error handling
    - Test all error scenarios
    - Test error responses sent to clients
    - Test error logging
    - Test metrics updates on errors
    - Test recovery from errors
    - _Requirements: 7.3, 7.6, 7.7, 11.1_

- [ ] 19. Implement at-most-once delivery mode
  - [ ] 19.1 Add delivery guarantee configuration
    - Support per-message-type delivery guarantees
    - Configure at-most-once vs at-least-once
    - _Requirements: 10.6_

  - [ ] 19.2 Implement at-most-once delivery path
    - Skip message store persistence
    - Skip acknowledgment tracking
    - Deliver once without retry
    - Discard on failure
    - _Requirements: 10.1, 10.2, 10.3, 10.4_

  - [ ]* 19.3 Write property test for at-most-once delivery count
    - **Property 35: At-Most-Once Delivery Count**
    - **Validates: Requirements 10.1**

  - [ ]* 19.4 Write unit tests for at-most-once mode
    - Test no persistence for at-most-once messages
    - Test no acknowledgment wait
    - Test discard on failure
    - Test latency (<10ms p99)
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

- [ ] 20. Implement message persistence and recovery
  - [ ] 20.1 Implement broker restart recovery
    - Load unacknowledged messages from store
    - Restore sequence numbers per client
    - Complete recovery within 5 seconds
    - Maintain message ordering
    - _Requirements: 11.2, 11.3_

  - [ ] 20.2 Implement historical message retention
    - Retain messages for 24 hours after session end
    - Support querying by session ID and sequence range
    - _Requirements: 11.5, 11.6_

  - [ ]* 20.3 Write property test for recovery ordering preservation
    - **Property 41: Recovery Ordering Preservation**
    - **Validates: Requirements 11.3**

  - [ ]* 20.4 Write unit tests for recovery
    - Test recovery loads unacknowledged messages
    - Test recovery timing (<5 seconds)
    - Test ordering preserved after recovery
    - Test historical message queries
    - Test 24-hour retention policy
    - _Requirements: 11.2, 11.3, 11.5, 11.6_

- [ ] 21. Integrate with existing broker
  - [ ] 21.1 Extend broker.Server to support sessions
    - Add session management to Server
    - Create sessions on client join
    - Route messages to appropriate session
    - Handle session lifecycle
    - _Requirements: 1.1_

  - [ ] 21.2 Extend broker.Client to support delivery guarantees
    - Add reconnection token handling
    - Add acknowledgment sending
    - Add reorder buffer
    - Handle snapshots and deltas
    - _Requirements: 1.3, 3.4, 4.1, 6.1_

  - [ ] 21.3 Maintain backward compatibility
    - Support clients without delivery guarantees
    - Feature negotiation on connection
    - Graceful degradation for old clients
    - _Requirements: 1.1_

  - [ ]* 21.4 Write integration tests for broker integration
    - Test session creation and management
    - Test message routing to sessions
    - Test backward compatibility with old clients
    - Test multiple sessions concurrently
    - _Requirements: 1.1_

- [ ] 22. Checkpoint - Full integration complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 23. Performance optimization and benchmarking
  - [ ] 23.1 Run benchmark tests
    - Benchmark message delivery latency (target: <50ms p99)
    - Benchmark throughput (target: >300 msg/s for 10 clients)
    - Benchmark broadcast to 10 clients (target: <5ms)
    - Benchmark message validation (target: <1ms)
    - Benchmark snapshot generation (target: <100ms for 10MB)
    - Benchmark delta generation (target: <10ms)
    - Benchmark at-most-once latency (target: <10ms p99)
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 10.5_

  - [ ] 23.2 Optimize performance bottlenecks
    - Profile CPU and memory usage
    - Optimize hot paths identified by benchmarks
    - Reduce allocations in critical paths
    - Optimize lock contention
    - _Requirements: 9.1, 9.2_

  - [ ]* 23.3 Write benchmark tests
    - Create benchmark suite for all performance requirements
    - Test with 2, 5, and 10 concurrent clients
    - Test with various message rates (10, 20, 30 msg/s per client)
    - Measure latency percentiles (p50, p95, p99)
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7_

- [ ] 24. Documentation and examples
  - [ ] 24.1 Create API documentation
    - Document Session configuration options
    - Document message protocol extensions
    - Document error codes and handling
    - Document metrics and monitoring
    - _Requirements: All_

  - [ ] 24.2 Create usage examples
    - Example: Basic session with delivery guarantees
    - Example: Reconnection after disconnect
    - Example: State snapshots and delta updates
    - Example: Rate limiting and validation
    - Example: Metrics collection and monitoring
    - _Requirements: All_

  - [ ] 24.3 Update README files
    - Update broker/README.md with delivery guarantees
    - Add migration guide for existing users
    - Add troubleshooting section
    - _Requirements: All_

- [ ] 25. Final checkpoint - Implementation complete
  - Ensure all tests pass, ask the user if questions arise.
  - Verify all 51 correctness properties are tested
  - Verify all performance benchmarks meet targets
  - Verify documentation is complete

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Property tests validate universal correctness properties (51 total)
- Unit tests validate specific examples and edge cases
- Integration tests validate component interactions
- Benchmark tests validate performance requirements
- Implementation uses Go and builds on existing broker and tcp-conn packages
- All components designed for 2-10 clients, 10-30 msg/s per client
- Target performance: <50ms p99 latency, 300 msg/s throughput
