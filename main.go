package main

import (
	_ "embed"

	"github.com/plingatech/go-plgchain/command/root"
	"github.com/plingatech/go-plgchain/licenses"
)

var (
	//go:embed LICENSE
	license string
)

func main() {
	licenses.SetLicense(license)

	root.NewRootCommand().Execute()
}
