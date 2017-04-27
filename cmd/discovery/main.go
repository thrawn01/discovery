package main

import (
	"fmt"
	"os"

	"github.com/mailgun/discovery"
	"github.com/thrawn01/args"
)

func checkErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	parser := args.NewParser()
	parser.AddArgument("service").Help("The name of the service to lookup")
	parser.AddArgument("port").Default("client").Help("The name of the port to lookup")
	parser.AddArgument("net").Choices([]string{"tcp", "udp"}).Default("tcp").
		Help("The name of the network to lookup")

	opts := parser.ParseOrExit(nil)

	results, err := discovery.Services(
		opts.String("service"),
		opts.String("port"),
		opts.String("net"),
		"Target: {{.Target}} Port: {{.Port}}")
	checkErr(err)

	fmt.Println("# Results")
	for _, row := range results {
		fmt.Println(row)
	}
	os.Exit(0)
}
