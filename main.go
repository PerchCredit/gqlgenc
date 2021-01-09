package main

import (
	"context"
	"fmt"
	"os"

	"github.com/99designs/gqlgen/api"
	"github.com/perchcredit/gqlgenc/clientgen"
	"github.com/perchcredit/gqlgenc/config"
	"github.com/perchcredit/gqlgenc/generator"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err.Error())
		os.Exit(2)
	}

	clientPlugin := clientgen.New(cfg.Query, cfg.Client, cfg.Generate)
	if err := generator.Generate(ctx, cfg, api.AddPlugin(clientPlugin)); err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err.Error())
		os.Exit(4)
	}
}
