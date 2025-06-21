package internal

import (
	"sync"
	"github.com/gorilla/websocket"
)

type Message struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Content  string `json:"content"`
	Time     string `json:"time"`
}

type Client struct {
	Conn  *websocket.Conn
	Mutex sync.Mutex
}

func (c *Client) SafeWriteJSON(v interface{}) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.Conn.WriteJSON(v)
}
