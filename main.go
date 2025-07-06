package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
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
		msg, err := reader.ReadString('\n')
		if err != nil {
			c.Close()
			return
		}

		fmt.Printf("%s", msg)
		if _, err := c.Write([]byte(msg)); err != nil {
			c.Close()
			return
		}
	}

}
