package utils

import (
	"fmt"
	"testing"
)

func TestUDPConn(t *testing.T) {
	udpConn, err := NewUDPConn(":0")
	if err != nil {
		t.Fatal(err)
	}
	defer udpConn.Close()

	fmt.Println(udpConn.LocalAddr())
	fmt.Println(udpConn.RemoteAddr())
}
