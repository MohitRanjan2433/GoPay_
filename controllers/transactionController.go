package Controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"payment/database"
	models "payment/model"

	// "github.com/gorilla/mux"
	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const lettersandDigitss = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateTransactionId(length int) string{
	result := make([]byte, length)
	for i := range result{
		result[i] = lettersandDigitss[rng.Intn(len(lettersandDigitss))]
	}
	return string(result)
}

// CreateTransaction creates a new transaction and updates users' transaction lists
func CreateTransaction(transaction models.Transaction) error {
	transactionCollection := database.GetTransactionCollection()
	userCollection := database.GetUserCollection()

	// Ensure collections are initialized
	if transactionCollection == nil || userCollection == nil {
		return errors.New("Database collections are not initialized")
	}

	// Log the transaction values
	log.Printf("Transaction: %+v", transaction)

	// Check if the sender UPI exists and get their balance
	senderFilter := bson.M{"upi": transaction.From}
	var senderUser models.User

	log.Printf("Sender Filter: %v", senderFilter)
	err := userCollection.FindOne(context.TODO(), senderFilter).Decode(&senderUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("Sender UPI not found")
		}
		log.Printf("Error fetching sender: %v", err)
		return err
	}
	log.Printf("Sender User: %+v", senderUser)

	// Check if the receiver UPI exists
	receiverFilter := bson.M{"upi": transaction.To}
	var receiverUser models.User

	log.Printf("Receiver Filter: %v", receiverFilter)
	err = userCollection.FindOne(context.TODO(), receiverFilter).Decode(&receiverUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("Receiver UPI not found: %v", receiverFilter)
			return errors.New("Receiver UPI not found")
		}
		log.Printf("Error fetching receiver: %v", err)
		return err
	}
	log.Printf("Receiver User: %+v", receiverUser)

	// Check if the sender has enough balance
	if senderUser.Balance < float32(transaction.Amount) {
		return errors.New("Insufficient balance")
	}

	transaction.TransactionID = generateTransactionId(10)

	// Insert the transaction
	_, err = transactionCollection.InsertOne(context.TODO(), transaction)
	if err != nil {
		log.Printf("Error inserting transaction: %v", err)
		return err
	}

	// Update sender's transaction list and balance
	senderUpdate := bson.M{
		"$push": bson.M{"transactionId": transaction.TransactionID},
		"$inc":  bson.M{"balance": -transaction.Amount}, // Deduct amount from sender
	}
	_, err = userCollection.UpdateOne(context.TODO(), senderFilter, senderUpdate)
	if err != nil {
		log.Printf("Error updating sender: %v", err)
		return err
	}

	// Update receiver's transaction list and balance
	receiverUpdate := bson.M{
		"$push": bson.M{"transactionId": transaction.TransactionID},
		"$inc":  bson.M{"balance": transaction.Amount}, // Add amount to receiver
	}
	_, err = userCollection.UpdateOne(context.TODO(), receiverFilter, receiverUpdate)
	if err != nil {
		log.Printf("Error updating receiver: %v", err)
		return err
	}

	return nil
}


// CreateTransactionHandler handles HTTP requests for creating transactions
func CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }

    // Validate the JWT token
    token, err := validateToken(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        log.Printf("Error validating token: %s", err)
        return
    }

    // Extract email from token claims
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
        return
    }
    tokenEmail, ok := claims["email"].(string)
    if !ok {
        http.Error(w, "Invalid token claims", http.StatusUnauthorized)
        return
    }

    var transaction models.Transaction

    // Decode the request body into the Transaction model
    if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        log.Printf("Error decoding request body: %s", err)
        return
    }

	userCollection := database.GetUserCollection()
	var user models.User

	filter := bson.M{"email": tokenEmail}
	err = userCollection.FindOne(context.TODO(), filter).Decode(&user)

    // Ensure the sender in the transaction is the same as the user making the request
    if transaction.From != user.Upi {
        http.Error(w, "Unauthorized sender", http.StatusUnauthorized)
        log.Printf("Unauthorized sender attempts: %s", transaction.From)
        return
    }

    // Use safeCreateTransaction to handle panics
    err = safeCreateTransaction(transaction)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Send successful response
    response := map[string]string{"message": "Transaction created successfully"}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("Error encoding response: %s", err)
    }
}

func safeCreateTransaction(transaction models.Transaction) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Recovered from panic: %v", r)
        }
    }()

    err := CreateTransaction(transaction)
    if err != nil {
        log.Printf("Error creating transaction: %v", err)
        return err
    }
    return nil
}