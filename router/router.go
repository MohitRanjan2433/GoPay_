package routes

import (
	Controllers "payment/controllers"

	"github.com/gorilla/mux"
)

func SetupRoutes(r *mux.Router) *mux.Router{
	r.HandleFunc("/user", Controllers.CreateUserHandler).Methods("POST")
	r.HandleFunc("/login", Controllers.LoginUserHandler).Methods("POST")
	r.HandleFunc("/userdetails", Controllers.GetUserHandler).Methods("GET")
	r.HandleFunc("/transaction", Controllers.CreateTransactionHandler).Methods("POST")
	return r
}