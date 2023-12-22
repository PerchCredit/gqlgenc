module github.com/perchcredit/gqlgenc

go 1.15

require (
	github.com/99designs/gqlgen v0.17.20
	github.com/aws/aws-sdk-go v1.36.31
	github.com/dgryski/trifles v0.0.0-20200830180326-aaf60a07f6a3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.2
	github.com/vektah/gqlparser/v2 v2.5.10
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0
)

replace sourcegraph.com/sourcegraph/appdash => github.com/sourcegraph/appdash v0.0.0-20211028080628-e2786a622600

replace sourcegraph.com/sourcegraph/appdash-data => github.com/sourcegraph/appdash-data v0.0.0-20151005221446-73f23eafcf67
