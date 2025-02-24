package main

import (
	"flag"

	"github.com/0xfaulty/tcp-checker/internal/checker"
)

func main() {
	serverAddr := flag.String("addr", "127.0.0.1:7300", "Server address in format host:port")
	flag.Parse()

	checker.RunClient(*serverAddr)
}
