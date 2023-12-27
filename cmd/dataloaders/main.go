package main

import "github.com/SplitFi/go-splitfi/cmd/dataloaders/generator"

func main() {
	generator.Generate("./db/gen/coredb/manifest.json", "./graphql/dataloader")
}
