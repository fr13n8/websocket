package main

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.ListenAndServe(":8080", nil)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := NewWs(w, r)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ws.AcceptHandShake()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ws.Close()

	var message []byte
	for {
		header, err := ws.Read(2)
		if err != nil {
			fmt.Println(err)
			return
		}
		finFlag := (header[0] & 0x80) == 128
		optCode := header[0] & 0x0F
		maskFlag := (header[1] & 0x80) == 128
		length := uint64(header[1] & 0x7F)

		size := uint64(0)

		if length == 126 {
			data, err := ws.Read(2)
			if err != nil {
				fmt.Println(err)
				return
			}
			size = uint64(binary.BigEndian.Uint16(data))
		} else if length == 127 {
			data, err := ws.Read(8)
			if err != nil {
				fmt.Println(err)
				return
			}
			size = uint64(binary.BigEndian.Uint64(data))
		} else if length <= 125 {
			size = uint64(length)
		}

		payload := make([]byte, size)

		if maskFlag {
			key, err := ws.Read(4)
			if err != nil {
				fmt.Println(err)
				return
			}
			maskKey := key
			payload, err = ws.Read(int(size))
			if err != nil {
				fmt.Println(err)
				return
			}

			for i := uint64(0); i < size; i++ {
				payload[i] = payload[i] ^ maskKey[i%4]
			}
		}

		if payload == nil {
			payload, err = ws.Read(int(size))
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		message = append(message, payload...)

		if optCode == 8 {
			return
		} else if finFlag {
			fmt.Println(strings.TrimSuffix(string(message), "\n"))
			message = message[:0]
		}

	}
}
