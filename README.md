## Chat Backend

A lightweight, real-time WebSocket chat server written in Go, designed for multi-client communication. Built with simplicity, performance, and modularity in mind.

## Features

- ⚡ Real-time messaging with WebSockets
- 👥 Live user list updates
- ✍️ "Typing..." notifications
- 🧼 Username validation and duplicate prevention
- 🧠 Spam/cooldown protection
- 🐳 Easy Docker-based setup
- 🤖 GitHub Actions CI for build & test automation



## Tech Stack

- [Go (Golang)](https://golang.org/) — core backend logic
- [Gorilla WebSocket](https://github.com/gorilla/websocket) — for real-time messaging
- [Docker](https://www.docker.com/) — containerization
- [GitHub Actions](https://github.com/features/actions) — continuous integration



## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/tzeytun/chat-backend.git
cd chat-backend
````

### 2. Run with Docker

```bash
docker compose up --build
```

> The server will be available at [http://localhost:8080](http://localhost:8080)

### 3. Check if it’s working

```bash
curl http://localhost:8080/ping
# Should return: pong
```



## Project Structure

```
chat-backend/
├── internal/
│   ├── handler.go        # WebSocket connection & message handling
│   └── types.go          # Client and message type definitions
├── main.go               # Server entry point
├── Dockerfile            # Build instructions for container
├── docker-compose.yml    # Compose configuration for backend
└── .github/workflows/    # GitHub Actions CI workflow
```



## Testing

If test files (`*_test.go`) are present, they will be automatically run via the CI pipeline on each push.



## API Endpoints

| Endpoint | Description              |
| -------- | ------------------------ |
| `/ws`    | WebSocket connection URL |
| `/ping`  | Simple health check      |



## Contributing

Contributions are welcome! Feel free to submit pull requests or open issues to suggest improvements, fix bugs, or propose new features.



## License

This project is licensed under the MIT License — see the `LICENSE` file for details.

