package protocol

import (
	"bufio"
	"errors"
	"log"
	"strconv"
)

func Parse(reader *bufio.Reader) ([]string, error) {
	result := make([]string, 0)
	prefix, _, _ := reader.ReadLine()
	if len(prefix) == 0 {
		return nil, errors.New("empty input")
	}
	totalElements, err := strconv.Atoi(string(prefix[1:]))
	if err != nil {
		log.Print(err)
		return result, err
	}
	for i := 0; i < totalElements; i++ {
		data, _, err := reader.ReadLine()
		if prefix[0] == '*' && data[0] == '$' {
			if data == nil {
				return nil, errors.New("nil data")
			}
			if err != nil {
				log.Print(err)
				return nil, err
			}
			length, _ := strconv.Atoi(string(data[1:]))
			text, _, _ := reader.ReadLine()
			result = append(result, string(text[:length]))
		}
	}
	return result, nil
}

func SerializeSimpleString(s string) string {
	// var result []byte
	return "+" + s + "\r\n"

}

func SerializeBulkString(s string) string {
	return "$" + strconv.Itoa(len(s)) + "\r\n" + s + "\r\n"
}

func SerializeInteger(n int) string {
	return ":" + strconv.Itoa(n) + "\r\n"
}

func SerializeError(msg string) string {
	return "-ERR " + msg + "\r\n"
}

func SerializeNull() string {
	return "$-1\r\n"
}
