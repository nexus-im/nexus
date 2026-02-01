# Messaging Spec (v1)

## Transport
- WebSocket only (no HTTP send).
- Client connects: `ws://<host>/ws?token=<session_token>`.

## Message Types

### 1) Client → Server: send_message
```json
{
  "type": "send_message",
  "payload": {
    "conversation_id": "uuid",
    "content": "Hello world",
    "client_id": "optional-client-generated-id"
  }
}
```

### 2) Server → Client: message_delivered
```json
{
  "type": "message_delivered",
  "payload": {
    "message_id": "uuid",
    "conversation_id": "uuid",
    "sender_id": "uuid",
    "content": "Hello world",
    "sent_at": "2026-01-24T22:15:08Z",
    "client_id": "optional-client-generated-id"
  }
}
```

### 3) Server → Client: error
```json
{
  "type": "error",
  "payload": {
    "code": "invalid_payload|unauthorized|server_error",
    "message": "human readable error"
  }
}
```

## Behavior
- Server validates auth via session token at WS connect.
- `send_message`:
  - Required: `conversation_id`, `content`.
  - Server creates message record (if DB storage enabled) and broadcasts `message_delivered` to all members of `conversation_id` (including sender).
  - If DB storage not enabled yet, broadcast only (in-memory) and generate `message_id` server-side.
- `client_id` is echoed back for client-side de-dupe/ack.

## Conversation Creation

**URL:** `POST /api/conversations`
**Headers:** `Authorization: Bearer <session_token>` (or `X-Session-Token`)

### P2P
```json
{
  "type": "p2p",
  "user_id": "other_user_uuid"
}
```

### Group
```json
{
  "type": "group",
  "member_ids": ["user_uuid_1", "user_uuid_2"]
}
```

**Response (201 Created or 200 OK if existing):**
```json
{
  "conversation_id": "uuid",
  "created": true
}
```

## Data Model (if persisted)
- `messages`: `id`, `conversation_id`, `sender_id`, `content`, `created_at`
- `conversation_members`: `conversation_id`, `user_id`

## Validation
- `content` length max 2000 chars.
- Reject empty content.
