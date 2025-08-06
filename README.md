## 🚀 Quick Start

### Run the Server

```bash
go run main.go serve
```

> App will be available at: `http://localhost:8080`

---

## 🔐 Authentication

Use the `/api/login` endpoint to simulate login:

```json
POST /api/login
{
  "user_id": "user-123",
  "tenant_id": "tenant-abc"
}
```

Response:

```json
{ "token": "Bearer eyJhbGciOi..." }
```

Use this token in `Authorization` header (Swagger has 🔒 button for this).

---

## 🛠️ Core APIs

| Method | Endpoint                                   | Description                          |
|--------|--------------------------------------------|--------------------------------------|
| POST   | `/api/login`                               | Mock login, returns JWT              |
| POST   | `/api/tenants`                             | Create a new tenant + consumer       |
| DELETE | `/api/tenants/{id}`                        | Delete tenant and shutdown consumer  |
| PUT    | `/api/tenants/{id}/config/concurrency`     | Update worker concurrency per tenant |
| POST   | `/api/messages/{tenant_id}`                | Publish a message to a tenant queue  |
| GET    | `/api/messages?cursor=...`                 | Fetch paginated messages             |

---

## 🔄 Cursor Pagination

Pagination uses encoded `created_at|uuid` cursors. Example:

```
GET /api/messages?cursor=eyIxMjM0NTYiOiJ0YWctMTIzIn0=
```

---

## 📄 Swagger Docs

Start the server and visit:

```
http://localhost:8080/swagger/index.html
```

---

## ⚙️ Configuration

File: `config/config.yaml`

```yaml
server:
  port: 8080

database:
  url: postgres://postgres:postgres@localhost:5432/messaging?sslmode=disable

rabbitmq:
  url: amqp://guest:guest@localhost:5672/

jwtConfig:
  secret: your-secret-key

workers: 3
```

---

## 🧪 Running Tests

```go test -v --tags=integration ./...```

---

## 📌 Notes

- All RabbitMQ queues are dynamically created per tenant: `tenant_{id}_queue`
- PostgreSQL `messages` table is partitioned by `tenant_id`
- Message processing is fan-in to worker pool per tenant
- JWT token embeds `user_id` and `tenant_id`

---

## 🧑‍💻 Author

**Feka Legi**

---

## 📃 License

MIT
```
