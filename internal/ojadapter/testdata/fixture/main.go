package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "--stdio" {
		os.Exit(2)
	}
	s := bufio.NewScanner(os.Stdin)
	s.Buffer(make([]byte, 4096), 9<<20)
	for s.Scan() {
		var r struct {
			ID     uint64          `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(s.Bytes(), &r) != nil {
			os.Exit(3)
		}
		switch r.Method {
		case "capabilities":
			fmt.Printf("{\"id\":%d,\"result\":{\"protocol\":\"routilet-oj-adapter/1\",\"adapter\":\"fixture\",\"version\":\"0.0.0-dev\",\"methods\":[\"capabilities\",\"echo\"]}}\n", r.ID)
		case "echo":
			fmt.Printf("{\"id\":%d,\"result\":%s}\n", r.ID, r.Params)
		case "error":
			fmt.Printf("{\"id\":%d,\"error\":{\"code\":\"fixture_error\",\"message\":\"fixture\"}}\n", r.ID)
		case "hang":
			time.Sleep(time.Hour)
		case "crash":
			os.Exit(4)
		case "bad-id":
			fmt.Printf("{\"id\":%d,\"result\":{}}\n", r.ID+1)
		case "both":
			fmt.Printf("{\"id\":%d,\"result\":{},\"error\":{\"code\":\"x\",\"message\":\"x\"}}\n", r.ID)
		case "partial":
			fmt.Printf("{\"id\":%d,\"result\":{}}", r.ID)
			os.Exit(0)
		case "large":
			fmt.Println(strings.Repeat("x", (8<<20)+2))
		default:
			fmt.Println("not JSON")
		}
	}
}
