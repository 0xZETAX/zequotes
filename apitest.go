package main

import (
	_ "embed"
	"fmt"
)

//go:embed api/data/quotes.json
var quotesJSON []byte

func main() {
	fmt.Println(string(quotesJSON))
}
