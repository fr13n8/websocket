package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
)

type Ws struct {
	conn   net.Conn
	bufrw  *bufio.ReadWriter
	header http.Header
}

func NewWs(w http.ResponseWriter, r *http.Request) (*Ws, error) {
	hjck, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("webserver doesn't support hijacking")
	}
	conn, bufrw, err := hjck.Hijack()
	if err != nil {
		return nil, err
	}
	return &Ws{
		conn:   conn,
		bufrw:  bufrw,
		header: r.Header,
	}, nil
}

func (ws *Ws) Read(size int) ([]byte, error) {
	data := make([]byte, 0)
	for {
		if len(data) == size {
			break
		}

		nextSize := size - len(data)
		chunk := make([]byte, nextSize)

		n, err := ws.bufrw.Read(chunk)
		if err != nil && err != io.EOF {
			return nil, err
		}

		data = append(data, chunk[:n]...)
	}
	return data, nil
}

func (ws *Ws) AcceptHandShake() error {
	err := ws.validateHandShake()
	if err != nil {
		return err
	}

	ws.bufrw = ws.writeHandShake()
	return nil
}

func (ws *Ws) validateHandShake() error {
	if ws.header.Get("Upgrade") != "websocket" {
		return errors.New("upgrade header is not websocket")
	}

	if ws.header.Get("Connection") != "Upgrade" {
		return errors.New("connection header is not Upgrade")
	}

	if ws.header.Get("Sec-WebSocket-Key") == "" {
		return errors.New("Sec-WebSocket-Key is empty")
	}

	return nil
}

func (ws *Ws) getAcceptHashKey() string {
	keySum := ws.header.Get("Sec-WebSocket-Key") + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hash := sha1.Sum([]byte(keySum))
	b64str := base64.StdEncoding.EncodeToString(hash[:])
	return b64str
}

func (ws *Ws) writeHandShake() *bufio.ReadWriter {
	acceptHashKey := ws.getAcceptHashKey()
	ws.bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	ws.bufrw.WriteString("Upgrade: websocket\r\n")
	ws.bufrw.WriteString("Connection: Upgrade\r\n")
	ws.bufrw.WriteString("Sec-Websocket-Accept: " + acceptHashKey + "\r\n\r\n")
	ws.bufrw.Flush()
	return ws.bufrw
}

func (ws *Ws) Close() error {
	return ws.conn.Close()
}
