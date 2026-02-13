The socket package must be redesigned around a server-side connected client registry that manages multiple concurrent logical clients.

Each connected client must maintain two independent, strictly ordered frame queues based on frame type:

Request queue — contains request frames from other clients in the same channel.

Response queue — contains response frames to be sent to that specific client.

Both queues must enforce strict FIFO ordering. Ordering guarantees are isolated per client and must not be affected by other clients.

When a client invokes Send() to issue a request, the server must enqueue a request frame into all other clients' request queues within the same channel. If no other clients exist in the channel, the server must enqueue the request frame into the sending client's own request queue.

When a client invokes Receive(), the server must dequeue the next request frame from that client's own request queue. If the client's request queue is empty, the server must dequeue from other clients' request queues in the same channel.

After processing, when the client invokes Respond(), the server must enqueue the resulting response frame into that same client's response queue.

Response frames must be transmitted to the originating client in the exact order they were enqueued.

Under no circumstances may request or response frames belonging to different clients be intermixed or share per-client queue state.
