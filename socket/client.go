package socket

import (
	"net"
)

type Client struct {
	conn net.Conn
}

func NewClient(port string) *Client {
	conn, _ := net.Dial("tcp", "localhost:"+port)
	return &Client{conn}
}

func (c *Client) Send(utf8Data string) error {
	_, err := c.conn.Write([]byte(utf8Data))
	return err
}

func (c *Client) Receive() (string, error) {
	buffer := make([]byte, 1024)
	n, err := c.conn.Read(buffer)
	if err != nil {
		return "", err
	}
	return string(buffer[:n]), nil
}

func (c *Client) Close() error {
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) IsConnected() bool {
	return c.conn != nil
}
