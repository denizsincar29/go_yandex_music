package main

import (
	"flag"
	"log"
)

func main() {
	port := flag.String("port", "8080", "Port to run the web server on")
	flag.Parse()

	err := StartWebServer(*port)
	if err != nil {
		log.Fatal(err)
	}
}
