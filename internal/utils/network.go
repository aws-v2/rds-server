package utils

import (
	"fmt"
	"net"
	"time"
)

// CheckReachability performs a TCP reachability check on the given host and port.
func CheckReachability(host string, port interface{}, timeout time.Duration) error {
	address := fmt.Sprintf("%s:%v", host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
