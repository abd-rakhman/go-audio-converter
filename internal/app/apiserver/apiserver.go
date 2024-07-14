package apiserver

import (
	"log"
	"net/http"
)

func Start(config AppConfig) error {
	srv := newServer(config.FORWARD_URL)

	log.Printf("Server is listening on port %s...\n", config.BIND_ADDRESS)
	port := ":" + config.BIND_ADDRESS
	return http.ListenAndServe(port, srv)
}
