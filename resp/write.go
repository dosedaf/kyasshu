package resp

import (
	"fmt"
	"net"
)

func WriteInteger(c net.Conn, val int) {
	resp := fmt.Sprintf(":%d\r\n", val)
	c.Write([]byte(resp))
}

func WriteBulkString(c net.Conn, val string) {
	resp := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
	c.Write([]byte(resp))
}

func WriteNullBulkString(c net.Conn) {
	c.Write([]byte("$-1\r\n"))
}

func WritePONG(c net.Conn) {
	c.Write([]byte("+PONG\r\n"))
}

func WriteOK(c net.Conn) {
	c.Write([]byte("+OK\r\n"))
}

func WriteERR(c net.Conn, msg string) {
	resp := fmt.Sprintf("-ERR %s\r\n", msg)
	c.Write([]byte(resp))
}

func WriteNull(c net.Conn) {
	c.Write([]byte(""))
}

func SerializeCommand(cmd []string) []byte {
	var result []byte

	result = append(result, []byte(fmt.Sprintf("*%d\r\n", len(cmd)))...)

	for _, part := range cmd {
		result = append(result, []byte(fmt.Sprintf("$%d\r\n", len(part)))...)

		result = append(result, []byte(part)...)

		result = append(result, []byte("\r\n")...)
	}

	return result
}
