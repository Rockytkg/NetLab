# API Change Notes

Updated: 2026-07-07

Swagger was regenerated locally with:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go --output docs
```

Generated files:

- `NetLab-backend/docs/docs.go`
- `NetLab-backend/docs/swagger.json`
- `NetLab-backend/docs/swagger.yaml`

Note: `NetLab-backend/docs/` is currently ignored by `NetLab-backend/.gitignore`, so these generated Swagger artifacts are local unless force-added or the ignore rule is changed.

## Account APIs

### Send change-email code

`POST /api/auth/account/email-change-code`

Authentication: Bearer token required.

Request:

```json
{
  "newEmail": "new-user@example.com"
}
```

Behavior:

- Validates non-empty email, email format, length, and uniqueness.
- Sends a 6-digit verification code to the new email.
- Code purpose is `change-email`.
- Code validity is 5 minutes.

### Change account email

`PUT /api/auth/account/email`

Authentication: Bearer token required.

Request:

```json
{
  "newEmail": "new-user@example.com",
  "verifyCode": "123456"
}
```

Behavior:

- Validates email format and uniqueness.
- Validates the 6-digit code sent to the new email.
- Returns the updated current-user profile.

## Admin User APIs

### List users

`GET /api/admin/users`

Authentication: Bearer token with `admin` role required.

Query parameters:

- `page`: 1-based page number.
- `size`: page size.
- `keyword`: fuzzy match username or email.
- `status`: `active`, `disabled`, or `locked`.
- Administrator-role users are returned and managed like other users; the built-in `super_admin` bootstrap account remains hidden from the standard list.

### Update one user

`PUT /api/admin/users/{id}`

Request:

```json
{
  "email": "user@example.com",
  "roles": ["viewer"],
  "status": "active"
}
```

Administrator-role users can be edited through this endpoint.

### Batch update roles

`PUT /api/admin/users/roles`

Request:

```json
{
  "userIds": ["uuid"],
  "roles": ["viewer"]
}
```

Only `editor` and `viewer` are assignable through the management UI/API.

### Batch update emails

`PUT /api/admin/users/emails`

Request:

```json
{
  "updates": [
    {
      "userId": "uuid",
      "email": "new-user@example.com"
    }
  ]
}
```

Each email is validated for format and uniqueness before write.

### Batch delete users

`DELETE /api/admin/users`

Request:

```json
{
  "userIds": ["uuid"]
}
```

Administrator-role users can be deleted through this endpoint.

### Import users

`POST /api/admin/users/import`

Multipart form-data:

- `file`: CSV file, max 2 MB.

CSV headers:

- `username`: required, 3-64 characters, letters/numbers/underscore/hyphen.
- `email`: required, valid and unique.
- `role`: optional, `viewer` or `editor`, defaults to `viewer`.
- `password`: optional, defaults to username, must pass password strength validation.
