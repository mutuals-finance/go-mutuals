package main

import (
	_ "net/http/pprof"

	"github.com/SplitFi/go-splitfi/indexer/cmd"
)

func main() {
	cmd.Execute()
}
