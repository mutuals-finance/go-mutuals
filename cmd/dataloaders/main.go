package main

import "github.com/mutuals/go-mutuals/cmd/dataloaders/generator"

func main() {
	generator.Generate("./db/gen/coredb/manifest.json", "./graphql/dataloader")
}
