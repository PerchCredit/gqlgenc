package generator

import (
	"context"
	"log"
	"testing"

	"github.com/perchcredit/gqlgenc/config"
)

func TestLoadConfig(t *testing.T) {

	config, err := config.LoadConfig("../config/testdata/cfg/endpoint.yml")
	if err != nil {
		log.Fatal(err)
	}

	err = Generate(context.Background(), config)
	if err != nil {
		log.Fatal(err.Error())
	}
}
