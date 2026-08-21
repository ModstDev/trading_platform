<img width="951" height="468" alt="TradePlat" src="https://github.com/user-attachments/assets/03337667-e455-4083-bfd6-6e94a56a0abf" />

# Trading Platform

A simulated brokerage/trading platform built with Go.

The project is designed as a backend-focused learning project using a production-style architecture. It combines a REST API, MariaDB, NATS JetStream, background workers, an order-matching engine and real market-price data.

> **Important:** This platform does not execute real trades or use real money.
> Trading activity is simulated. Real market data is used only to provide
> current market prices.
> THE PROJECT IS STILL UNDER DEVELOPMENT!

---

## Features

### Authentication

- User registration
- User login
- JWT authentication
- UUID-based user IDs
- Password hashing

### Accounts

- Account creation
- Account balance
- Reserved balance
- Available balance
- Account information

### Instruments

The platform supports financial instruments such as stocks.

Each instrument contains information such as:

- Symbol
- Name
- Type
- Currency

### Orders

Users can create:

- BUY orders
- SELL orders
- LIMIT orders
- MARKET orders

Users can also:

- View their orders
- Cancel pending orders
- Track filled quantities
- View order status

### Matching Engine

Orders are processed asynchronously by a dedicated matching worker.

The matching engine:

- Finds compatible buy and sell orders
- Uses price-time priority
- Locks relevant database rows when necessary
- Creates executions
- Updates orders
- Updates positions
- Updates account balances

All executions are simulated.

### Positions

Users can view their current holdings, including:

- Instrument
- Quantity
- Reserved quantity
- Average price

### Executions

Executed trades are recorded and can be viewed by the user.

### Real Market Data

The platform receives real market prices from Twelve Data.

For example:

```text
AAPL -> $226.45
MSFT -> $505.12
