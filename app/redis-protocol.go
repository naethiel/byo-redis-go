package main

import (
	"bufio"
	"fmt"
	"strconv"
)

func encodeSimpleString(s []byte) []byte {
	return []byte(fmt.Sprintf("+%s\r\n", s))
}

func encodeBulkString(s []byte) []byte {
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(s), s))
}

func decodeSimpleString(stream *bufio.Reader) (RESP, error) {
	val, err := readToCRLF(stream)
	if err != nil {
		return RESP{}, err
	}
	return RESP{
		Typ:   SimpleString,
		Value: val,
	}, nil
}

func decodeBulkString(stream *bufio.Reader) (RESP, error) {
	rawSize, err := readToCRLF(stream)
	if err != nil {
		return RESP{}, fmt.Errorf("reading size of resp bulk string: %w", err)
	}

	size, err := strconv.Atoi(string(rawSize))
	if err != nil {
		return RESP{}, fmt.Errorf("converting size of resp bulk string to int: %w", err)
	}

	if size == -1 {
		// special null bulk string
		return RESP{
			Typ:   BulkString,
			Value: []byte{},
		}, nil
	}

	value, err := readToCRLF(stream)
	if err != nil {
		return RESP{}, fmt.Errorf("reading bulk string content: %w", err)
	}

	return RESP{
		Typ:   BulkString,
		Value: value,
	}, nil
}

func decodeArray(stream *bufio.Reader) (RESP, error) {
	rawSize, err := readToCRLF(stream)
	if err != nil {
		return RESP{}, fmt.Errorf("reading len of resp array: %w", err)
	}

	length, err := strconv.Atoi(string(rawSize))
	if err != nil {
		return RESP{}, fmt.Errorf("converting length of resp arr to int: %w", err)
	}

	base := RESP{
		Typ:    Array,
		Length: length,
	}
	children := make([]RESP, 0, length)

	for i := 0; i < base.Length; i++ {
		item, err := decodeRESP(stream)
		if err != nil {
			fmt.Println("error while decoding resp item in resp array: ", err.Error())
			break
		}
		children = append(children, item)
	}

	return RESP{
		Typ:      Array,
		Children: children,
		Length:   length,
	}, nil
}

func decodeRESP(stream *bufio.Reader) (RESP, error) {
	if stream.Size() == 0 {
		// TODO return a null resp
		return RESP{}, nil
	}

	typ, err := stream.ReadByte()
	if err != nil {
		return RESP{}, err
	}

	switch RESPType(typ) {
	case SimpleString:
		return decodeSimpleString(stream)
	case BulkString:
		return decodeBulkString(stream)
	case Array:
		return decodeArray(stream)
	default:
		return RESP{}, fmt.Errorf("unhandled RESP type: %s", string(typ))
	}
}

func readToCRLF(stream *bufio.Reader) ([]byte, error) {
	val, err := stream.ReadBytes('\n')
	if err != nil {
		return nil, wrapErr(err, "reading to CRLF")
	}
	if val[len(val)-2] != '\r' {
		return nil, fmt.Errorf("missing CR char in %s", string(val))
	}

	return val[:len(val)-2], nil
}

type RESPType byte

const (
	SimpleString RESPType = '+'
	Error        RESPType = '-'
	Integer      RESPType = ':'
	BulkString   RESPType = '$'
	Array        RESPType = '*'
)

type RESP struct {
	Typ      RESPType
	Value    []byte
	Length   int
	Children []RESP
}
