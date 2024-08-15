package models

type User struct {
	Key         string   `json:"key" bson:"key" validate:"required, min=2, max=10"`
	FirstName   string   `json:"firstName" bson:"firstName" validate:"required"`
	LastName    string   `json:"lastName" bson:"lastName" validate:"required"`
	Email       string   `json:"email" bson:"email" validate:"required"`
	Password    string   `json:"password" bson:"password" validate:"required,min=2, max=10"`
	Upi         string   `json:"upi" bson:"upi" validate:"required"`
	Balance     float32	 `json:"balance" bson:"balance" validate:"required"`
	TransactionID []string `json:"transactionId" bson:"transactionId"` // References to transactions
}

type UserResponse struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Upi       string `json:"upi"`
}

type Transaction struct {
	TransactionID   string  `json:"transactionId" bson:"transactionId" validate:"required"`
	Amount          float64 `json:"amount" bson:"amount" validate:"required"`
	TransactionDate string  `json:"transactionDate" bson:"transactionDate" validate:"required"`
	From            string  `json:"fromUpi" bson:"from" validate:"required"`
	To              string  `json:"toUpi" bson:"to" validate:"required"`
}
