# Trader frontend

This version was built against the current API in `ModstDev/trading_platform`.

It intentionally uses only endpoints currently exposed by the backend:

- POST /users
- POST /login
- GET /me
- GET /account
- GET /instruments
- POST /orders
- GET /orders
- DELETE /orders/{id}
- GET /positions
- GET /executions

It does NOT use order-book, or manual matching endpoints.

## Run

From this directory:

```bash
python3 -m http.server 3000
```

Then open:

http://localhost:3000

The API must be running at:

http://localhost:8080

## Important

The current backend's `/account` response exposes:

- id
- user_id
- balance
- currency

It does not expose `reserved_balance`, so the frontend does not invent or display a reserved value.

The current `/login` response contains `access_token`, and the frontend stores it in localStorage.

The current order model serializes nullable `price` and `max_cost` as `sql.NullString`; the frontend handles those values safely.
