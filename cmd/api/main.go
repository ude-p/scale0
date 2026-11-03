package main

import (
	"log"

	"github.com/ude-p/shipl/internal/deployments"
	"github.com/ude-p/shipl/internal/server"
)

func main() {
	deploymentService := deployments.NewService()
	deploymentHandler := deployments.NewHandler(deploymentService)

	srv := server.New(":8080", deploymentHandler)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
