package main

import (
	"fmt"
	"payment/router"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load();

	if err != nil{
		log.Fatal(err);
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("Error while getting port");
	}

	r := mux.NewRouter();

	// Setup routes
	routes.SetupRoutes(r)

	fmt.Println("Server is staring")
	fmt.Println("Server is listening on port", port);
	log.Fatal(http.ListenAndServe(":" + port, r));

}