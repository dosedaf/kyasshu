package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/dosedaf/kyasshu/resp"
)

type valueEntry struct {
	value     string
	expiresAt time.Time
}

type dataStore struct {
	mtx  sync.Mutex
	data map[string]valueEntry
}

var ds = dataStore{
	data: make(map[string]valueEntry),
}

func main() {
	l, err := net.Listen("tcp4", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	reader := bufio.NewReader(c)
	defer c.Close()

	for {
		cmd, err := resp.Parse(reader)
		if err != nil {
			log.Print(err)
			return
		}

		switch cmd[0] {
		case "PING":
			c.Write([]byte("+PONG\r\n"))
		case "SET":
			ds.mtx.Lock()
			ds.data[cmd[1]] = valueEntry{
				value: cmd[2],
			}

			ds.mtx.Unlock()

			c.Write([]byte("+OK\r\n"))
		case "GET":
			ds.mtx.Lock()

			val, ok := ds.data[cmd[1]]
			ds.mtx.Unlock()

			if !ok {
				c.Write([]byte("$-1\r\n"))
			} else {
				if !val.expiresAt.IsZero() && !time.Now().Before(val.expiresAt) {
					ds.mtx.Lock()
					delete(ds.data, cmd[1])
					ds.mtx.Unlock()
					c.Write([]byte("$-1\r\n"))
				} else {
					resp := fmt.Sprintf("$%d\r\n%s\r\n", len(val.value), val.value)
					c.Write([]byte(resp))
				}
			}
		case "EXPIRE":
			ds.mtx.Lock()

			val, ok := ds.data[cmd[1]]

			ds.mtx.Unlock()

			if !ok {
				c.Write([]byte(":0\r\n"))
			} else {
				sec, err := strconv.Atoi(cmd[2])
				if err != nil {
					c.Write([]byte(":0\r\n"))
				} else {
					timein := time.Now().Local().Add(time.Second * time.Duration(sec))

					ds.mtx.Lock()

					ds.data[cmd[1]] = valueEntry{
						value:     val.value,
						expiresAt: timein,
					}

					ds.mtx.Unlock()

					c.Write([]byte(":1\r\n"))
				}

			}
		case "TTL":
			ds.mtx.Lock()

			val, ok := ds.data[cmd[1]]
			ds.mtx.Unlock()

			fmt.Println("here")

			if !ok {
				c.Write([]byte(":-2\r\n"))
			} else {
				if val.expiresAt.IsZero() {
					c.Write([]byte(":-1\r\n"))
				} else if !time.Now().Before(val.expiresAt) {
					c.Write([]byte(":-2\r\n"))
				} else {
					remainingSeconds := time.Until(val.expiresAt).Seconds()
					resp := fmt.Sprintf(":%d\r\n", int(remainingSeconds))
					c.Write([]byte(resp))
				}
			}

		default:
			c.Write([]byte("-ERR unknown command\r\n"))
		}

		fmt.Println(cmd)
	}
}
