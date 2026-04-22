# Chirpy

A RESTful API for a simple social platform where users can create and manage short messages ("chirps"). Built in Go with PostgreSQL, featuring authentication, authorization, and webhook-driven upgrades.

---

## Features

- User authentication (JWT)
- Secure password hashing (Argon2)
- Refresh tokens for session management
- Create, view, and delete chirps
- Filter and sort chirps via query parameters
- Authorization (users can only modify their own data)
- Webhook integration (Polka payments)
- "Chirpy Red" premium feature system

---

## Motivation

Chirpy was built to practice building a real-world backend API using:

- RESTful design principles
- Database migrations
- Authentication and authorization
- External integrations (webhooks)

---

## Tech Stack

- **Language:** Go
- **Database:** PostgreSQL
- **ORM/Queries:** SQLC
- **Migrations:** Goose
- **Auth:** JWT + Argon2
- **Environment:** .env

---

## Installation

### 1. Clone the repo

```bash
git clone https://github.com/ShiriezH/Chirpy
cd Chirpy
```
---

### 2. Install dependencies
```bash
go mod tidy
```
---

### 3. Create a .env file
```env
DB_URL=postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable
JWT_SECRET=your_secret_here
POLKA_KEY=your_polka_key_here
PLATFORM=dev
```
---

### 4. Create the database
```bash
createdb chirpy
```
---

### 5. Run database migrations 
```bash
source .env
goose -dir sql/schema postgres "$DB_URL" up
```
--- 

### 6. Generate SQL code
```bash
sqlc generate
```
---

### 7. Run the server
```bash
go run .
```
- Server runs on: http://localhost:8080

--- 

## API Overview

### Users

- POST /api/users → Create user
- PUT /api/users → Update user (auth required)

### Authentication

- POST /api/login → Login and receive tokens

###  Chirps

- POST /api/chirps → Create chirp (auth required)
- GET /api/chirps → Get all chirps
- GET /api/chirps?author_id=<uuid> → Filter by user
- GET /api/chirps?sort=asc|desc → Sort chirps
- GET /api/chirps/{chirpID} → Get single chirp
- DELETE /api/chirps/{chirpID} → Delete chirp (owner only)

### Tokens

- POST /api/refresh → Get new access token
- POST /api/revoke → Revoke refresh token

###  Webhooks

- POST /api/polka/webhooks → Handle payment events

### Security

- Passwords are hashed using Argon2
- JWT tokens are securely signed
- Webhooks are protected with an API key
- Users can only modify their own data

--- 

## Project Structure
```
Chirpy/
├── internal/
│   ├── auth/
│   └── database/
├── sql/
│   ├── queries/
│   └── schema/
├── main.go
├── .env
├── go.mod
└── README.md
```
---

### Notes

This project was built as part of a guided course and extended with additional features to practice backend development concepts.

---