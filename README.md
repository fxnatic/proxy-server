# Proxy Middleman Server (proxy-server)

A lightweight and efficient proxy server designed to forward requests to a single destination server. Perfect for managing IP-bound http proxies across various devices.

## Features

- Authentication: Incoming requests are authenticated by validating the username and password.
- Rate Limiting: Option to limit the amount of data used per username.
- Thread-Safe Mapping: Thread-safe map to store and manage users.
- Request Logging: Log details about each request.

## Usage

1. Clone the repository:
    ```bash
    git clone https://github.com/fxnatic/proxy-server.git
    cd proxy-server
    ```

2. Install dependencies:
    ```bash
    go mod tidy
    ```

3. Create `proxies.txt` file and add your proxies to it (ip:port or ip:port:user:pass).

4. Run the server:
    ```bash
    go run main.go
    ```

## License

This project is licensed under the [MIT License](/LICENSE).