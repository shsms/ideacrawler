package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

type workerManager struct {
}

func joinServerCluster() {
	startClusterListener()
}

func startClusterListener() {
	listener, err := net.Listen("tcp", cliParams.ClusterListenAddress)
	if err != nil {
		log.Printf("failed to listen: %#v", err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("Listening on", cliParams.ClusterListenAddress)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept: %#v", err)
			continue
		}
		go createNewClient(conn)
	}
}

func createNewClient(conn net.Conn) {

}
