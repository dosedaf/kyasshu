package main

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/dosedaf/kyasshu/resp"
)

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
	fmt.Printf("serving %s\n", c.RemoteAddr().String())
	reader := bufio.NewReader(c)
	defer c.Close()

	for {
		n, err := resp.Parse(reader)
		if err != nil {
			log.Fatal(err)
			return
		}
		fmt.Println(n)
	}
}
