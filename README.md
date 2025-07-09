# Kyasshu (キャッシュ)

Kyasshu (Japanese for "Cache") is a simple, in-memory key-value store built from scratch in Go. It is inspired by Redis and implements the RESP (REdis Serialization Protocol) to handle commands concurrently.

This project was built as a deep dive into low-level networking, advanced concurrency patterns, protocol implementation, and software architecture in Go.

---

## ✨ Features
* **🚀 High-Performance Concurrent Architecture**:
    * The data store is sharded across multiple partitions, each protected by its own `RWMutex`, to allow for parallel command processing and minimize lock contention.
* **🔌 Concurrent TCP Server**: Handles multiple simultaneous clients using Goroutines.
* **📝 RESP Parser**: Decodes client commands from the raw network stream.
* **💾 AOF Persistence**: All write commands are logged to an Append-Only File, allowing the server's state to be recovered after a restart.
* **🧱 Modular Design**: Server logic, data storage, and protocol formatting are separated into distinct packages for clean, testable code.
* **🗂️ Data Structures**: Supports multiple data types with type-checking to prevent incorrect operations.
    * **Strings**: `PING`, `SET`, `GET`, `DEL`
    * **Expirations**: `EXPIRE`, `TTL`
    * **Lists**: `LPUSH`, `LPOP`
---

## 🚀 Getting Started

### Prerequisites

* Go 1.18 or newer.
* An optional but recommended Redis client like `redis-cli` for testing.

### Running the Server

1.  **Clone the repository:**
    ```sh
    git clone [https://github.com/dosedaf/kyasshu.git](https://github.com/dosedaf/kyasshu.git)
    cd kyasshu
    ```

2.  **Run the server:**
    ```sh
    go run .
    ```
    The server will start and listen on port `6379`.

---

## 💻 Usage

Once the server is running, open a **new terminal window** and connect using `redis-cli` or another Redis client.

```sh
# Connect to the Kyasshu server
redis-cli

# Start sending commands
127.0.0.1:6379> PING
PONG

127.0.0.1:6379> SET name "Kyasshu"
OK

127.0.0.1:6379> GET name
"Kyasshu"

127.0.0.1:6379> EXPIRE name 10
(integer) 1

127.0.0.1:6379> TTL name
(integer) 9

127.0.0.1:6379> DEL name
(integer) 1

127.0.0.1:6379> GET name
(nil)

```

## 🛣️ Future Work

This project is a foundation. The next major engineering challenges to tackle are:

* 🔬 **Benchmarking**: Write a comprehensive benchmark suite using Go's testing package to measure operations per second and validate the performance gains from the sharded architecture.

* 🗂️ **More Data Structures**: Add support for other Redis data types like Lists, Hashes, and Sets.

* **Advanced Persistence**: Implement snapshotting (RDB) as an alternative persistence strategy.


## 📜 License
This project is licensed under the MIT License.