package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

// Client communicates with the daemon over its Unix socket.
type Client struct {
	SocketPath string
}

// SendReload sends a legacy reload command.
func (c *Client) SendReload() error {
	conn, err := net.Dial("unix", c.SocketPath)
	if err != nil {
		return fmt.Errorf("connecting to daemon: %w", err)
	}
	defer func() { _ = conn.Close() }()
	_, err = conn.Write([]byte("reload\n"))
	return err
}

// Send sends a JSON request and reads the response.
func (c *Client) Send(req Request) (*Response, error) {
	conn, err := net.Dial("unix", c.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("connecting to daemon (is it running?): %w", err)
	}
	defer func() { _ = conn.Close() }()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("writing request: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}
		return nil, fmt.Errorf("no response from daemon")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &resp, nil
}

// ProcessUp tells the daemon to start processes from the given config.
// workDir is the caller's working directory, used as the default process dir.
func (c *Client) ProcessUp(configPath, workDir string) (*Response, error) {
	return c.Send(Request{Cmd: "process-up", ConfigPath: configPath, WorkDir: workDir})
}

// ProcessDown tells the daemon to stop processes from the given config.
func (c *Client) ProcessDown(configPath string) (*Response, error) {
	return c.Send(Request{Cmd: "process-down", ConfigPath: configPath})
}

// ProcessList asks the daemon for the status of all managed processes.
func (c *Client) ProcessList() (*Response, error) {
	return c.Send(Request{Cmd: "process-ls"})
}

// ProcessLogs asks the daemon for the log file path of a named process.
func (c *Client) ProcessLogs(name string) (*Response, error) {
	return c.Send(Request{Cmd: "process-logs", Name: name})
}
