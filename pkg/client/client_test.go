package client

import (
	"fmt"
	"testing"
)

func TestClientRun(t *testing.T) {
	client := &Client{}
	demo := func() []string {
		fmt.Println("demo test")
		return nil
	}
	client.Run(&demo)

}
