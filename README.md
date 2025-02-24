# TCP Checker 🔍

A lightweight TCP connection checker that detects disconnections **within 100ms** using minimal network traffic.

## 🚀 Features
- **Fast disconnect detection** (100ms).
- **Minimal network usage** (1 byte every 100ms).
- **Cross-platform** (Linux, macOS, Windows).
- **Configurable** via command-line flags.

## 📥 Installation
Requires **Go 1.18+**.

Clone the repository and build:
```sh
git clone https://github.com/0xfaulty/tcp-checker.git
cd tcp-checker
go build -o tcp-checker-client ./cmd/client
go build -o tcp-checker-server ./cmd/server
```

## 🏁 Usage

### **1️⃣ Start the server**
Run the TCP server on a specified port:
```sh
./tcp-checker-server -port 7300
```
Example output:
```
2025/02/24 12:00:33 Server is listening on port 7300
2025/02/24 12:00:42 Connection established from 192.168.1.2:52134
```

### **2️⃣ Start the client**
Run the TCP client to check the connection:
```sh
./tcp-checker-client -addr 192.168.1.10:7300
```
Example output:
```
2025/02/24 12:00:37 Connected to 192.168.1.10:7300 for the first time
2025/02/24 12:00:42 Connection with 192.168.1.10:7300 lost: EOF
2025/02/24 12:00:43 Connection to 192.168.1.10:7300 restored, downtime: 100ms
```

## ⚙️ Configuration

| Flag | Description | Default |
|------|------------|---------|
| `-port` | Server: TCP port to listen on | `7300` |
| `-addr` | Client: Server address (`host:port`) | `127.0.0.1:7300` |

## 🛠 How It Works
- The **server** sends **`\n` (newline) every 100ms**.
- The **client** waits for `\n`, using **`SetReadDeadline(100ms)`**.
- If **no `\n` arrives in 100ms**, the client **logs disconnection**.
- If the **server crashes, network drops, or firewall blocks**, the client **detects it instantly**.

## 🔥 Why Use This?
✅ **Faster than TCP Keep-Alive** (default TCP detection is 10+ sec).  
✅ **Lighter than WebSockets** (only 1 byte every 100ms).  
✅ **Ideal for server monitoring, failover detection, and debugging network issues**.

## 📜 License
MIT License. Free to use and modify.

---
Made with ❤️ by [0xfaulty](https://github.com/0xfaulty)

