# Trading Platform Test Frontend

Simple functional UI for the Go API.

Run the API:
```bash
go run ./cmd/api
```

Run the frontend from this directory:
```bash
python3 -m http.server 3000
```

Open `http://localhost:3000`.

Features:
- register
- login/logout
- account
- instruments
- create LIMIT/MARKET orders
- list orders
- cancel orders
- manually trigger matching
- positions
- executions

JWT is stored in localStorage.

If the browser reports a CORS error, add CORS support to the Go API for `http://localhost:3000`.
