package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/dosedaf/kyasshu/resp"
)

type dataStore struct {
	mtx  sync.Mutex
	data map[string]string
}

var ds = dataStore{
	data: make(map[string]string),
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
			ds.data[cmd[1]] = cmd[2]
			ds.mtx.Unlock()
			c.Write([]byte("+OK\r\n"))
		case "GET":
			ds.mtx.Lock()

			val, ok := ds.data[cmd[1]]
			ds.mtx.Unlock()

			if !ok {
				c.Write([]byte("$-1\r\n"))
			} else {
				resp := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
				c.Write([]byte(resp))
			}

		default:
			c.Write([]byte("-ERR unknown command\r\n"))
		}

		fmt.Println(cmd)
	}
}
