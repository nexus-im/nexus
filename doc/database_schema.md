# Database Schema Design

For this application, we will use a relational database structure. This schema is compatible with **SQLite** (for ease of development/embedded use) or **PostgreSQL** (for production).

## Users Table

The `users` table handles identity and authentication credentials.

**Table Name:** `users`

| Column Name | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` or `TEXT` | **PK**, Not Null | Unique identifier for the user. |
| `username` | `VARCHAR(50)` | **Unique**, Not Null | The display name used for login and chat. |
| `password_hash`| `VARCHAR(255)` | Not Null | The **bcrypt** hash of the user's password. *Never store plain text.* |
| `created_at` | `TIMESTAMP` | Default: `NOW()` | When the account was registered. |
| `last_seen` | `TIMESTAMP` | Nullable | Timestamp of the user's last activity/login. |

### SQL Definition (PostgreSQL Example)

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP WITH TIME ZONE
);

-- Index for fast lookups during login
CREATE INDEX idx_users_username ON users(username);
```

## Sessions Table

The `sessions` table stores authentication sessions for active logins.

**Table Name:** `sessions`

| Column Name | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | **PK**, Not Null | Unique identifier for the session. |
| `user_id` | `UUID` | **FK**, Not Null | References `users.id`. |
| `token` | `TEXT` | **Unique**, Not Null | Opaque session token. |
| `created_at` | `TIMESTAMP` | Default: `NOW()` | When the session was created. |
| `expires_at` | `TIMESTAMP` | Not Null | When the session expires. |

### SQL Definition (PostgreSQL Example)

```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
```

## Conversations Table

The `conversations` table defines chat threads between users.

**Table Name:** `conversations`

| Column Name | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | **PK**, Not Null | Unique identifier for the conversation. |
| `type` | `TEXT` | **Not Null** | `p2p` or `group`. |
| `created_by` | `UUID` | **FK**, Not Null | User who created the conversation. |
| `created_at` | `TIMESTAMP` | Default: `NOW()` | When the conversation was created. |

### Conversation Members Table

The `conversation_members` table links users to conversations.

**Table Name:** `conversation_members`

| Column Name | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `conversation_id` | `UUID` | **PK/FK**, Not Null | References `conversations.id`. |
| `user_id` | `UUID` | **PK/FK**, Not Null | References `users.id`. |
| `joined_at` | `TIMESTAMP` | Default: `NOW()` | When the user joined. |

### SQL Definition (PostgreSQL Example)

```sql
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL CHECK (type IN ('p2p', 'group')),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE conversation_members (
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_conversations_type ON conversations(type);
CREATE INDEX idx_conversation_members_user_id ON conversation_members(user_id);
```

### Go Struct Mapping (GORM)

If using GORM (Go Object Relational Mapper), the model would look like this:

```go
type User struct {
    ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"` 
    Username     string    `gorm:"uniqueIndex;not null;size:50"`
    PasswordHash string    `gorm:"not null"`
    CreatedAt    time.Time `gorm:"autoCreateTime"`
    LastSeen     time.Time
}
```
