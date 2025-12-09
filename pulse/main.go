package main

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/httpcontainer"
)

func main() {
	uri := containers.Get[contracts.URI](httpcontainer.HttpServerAddressId)
	host := "localhost"
	port := 3000
	uri.SetHost(&host)
	uri.SetPort(&port)
	server := containers.Get[contracts.HttpServer](httpcontainer.HttpServerId)
	server.Start()
}
