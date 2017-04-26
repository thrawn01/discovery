package main

import (
	"github.com/thrawn01/args"
	"github.com/mailgun/discovery"
	"fmt"
	"os"
)

func checkErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	parser := args.NewParser()
	parser.AddArgument("service-name").Help("The name of the service to lookup")

	opts := parser.ParseOrExit(nil)

	results, err := discovery.Services(opts.String("service-name"), "Target: {{.Target}} Port: {{.Port}}")
	checkErr(err)

	fmt.Println("# Results")
	for _, row := range results {
		fmt.Println(row)
	}
	os.Exit(0)
}
