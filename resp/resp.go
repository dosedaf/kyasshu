package resp

import (
	"bufio"
	"io"
	"log"
	"strconv"
	"strings"
)

func Parse(reader *bufio.Reader) ([]string, error) {
	dataType, err := reader.ReadByte()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	if dataType == '*' {
		return parseArray(reader)
	}

	return nil, err
}

func parseArray(reader *bufio.Reader) ([]string, error) {
	var results []string

	line, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	count, err := strconv.Atoi(strings.TrimSuffix(line, "\r\n"))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	for i := 0; i < count; i++ {
		elemType, err := reader.ReadByte()
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		if elemType == '$' {
			str, err := parseBulkStrings(reader)
			if err != nil {
				log.Fatal(err)
				return nil, err
			}

			results = append(results, str)
		}

	}

	return results, nil
}

func parseBulkStrings(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	len, err := strconv.Atoi(strings.TrimSuffix(line, "\r\n"))
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	payload := make([]byte, len)

	_, err = io.ReadFull(reader, payload)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	_, err = reader.Discard(2)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	return string(payload), nil
}
