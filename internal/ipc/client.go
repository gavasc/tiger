package ipc

import (
	"encoding/json"
	"net"
	"time"
)

// Client is a simple synchronous IPC client.
type Client struct {
	socketPath string
}

func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

func (c *Client) send(cmd Command) (Response, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, 2*time.Second)
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		return Response{}, err
	}

	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

func (c *Client) GetAll() ([]Item, error) {
	resp, err := c.send(Command{Op: OpGetAll})
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) Create(item Item) error {
	_, err := c.send(Command{Op: OpCreate, Item: item})
	return err
}

func (c *Client) Edit(item Item) error {
	_, err := c.send(Command{Op: OpEdit, Item: item})
	return err
}

func (c *Client) Delete(id string) error {
	_, err := c.send(Command{Op: OpDelete, ID: id})
	return err
}

func (c *Client) Start(id string) error {
	_, err := c.send(Command{Op: OpStart, ID: id})
	return err
}

func (c *Client) Pause(id string) error {
	_, err := c.send(Command{Op: OpPause, ID: id})
	return err
}

func (c *Client) Reset(id string) error {
	_, err := c.send(Command{Op: OpReset, ID: id})
	return err
}
