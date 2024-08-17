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
	r.HandleFunc("/getTransaction", Controllers.GetTransactionHandle).Methods("GET")
	r.HandleFunc("/getTransactionID/{id}", Controllers.GetTransactionByIDHandler).Methods("GET")
	r.HandleFunc("/updateKey", Controllers.UpdateKeyHandler).Methods("PUT")
	return r
}