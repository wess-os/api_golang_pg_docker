# CRUD API GOLANG

## Created using:
- Golang
- Mux
- PostgreSQL
- Docker
- Docker Compose

## How to execute app:
- docker compose up -d;

## Routes:
- GET /users
- GET /users/{id}
- POST /users
    - obs: "name" and "email" are required
- PUT /users/{id}
    - obs: "name" and "email" are required
- DELETE /users/{id}