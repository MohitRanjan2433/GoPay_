package Controllers

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"payment/database"
	models "payment/model"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

const lettersandDigits = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateRandomString(length int) string{
	result := make([]byte, length)
	for i := range result{
		result[i] = lettersandDigits[rng.Intn(len(lettersandDigits))]
	}
	return string(result)
}	

func generateUpi(length int) string{
	result := make([]byte, length)
	for i := range result{
		result[i] = lettersandDigits[rng.Intn(len(lettersandDigits))]
	}
	return string(result) + "@gopay"
}

func hashPassword(password string) (string, error){
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil{
		return "", err
	}
	return string(hashedPass), nil
}

func verifyPassword(hashedPassword, password string) bool{
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

var jwtSecret = []byte("gopay")

func generateToken(email string) (string ,error){
	claims := jwt.MapClaims{}
	claims["email"] = email
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func validateToken(r *http.Request) (*jwt.Token, error) {
	// Extract the token from the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("Missing Authorization header")
	}

	// Check if the Authorization header starts with "Bearer "
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return nil, errors.New("Invalid Authorization header format")
	}

	// Extract the token part
	tokenString := authHeader[7:]

	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure that the token's signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	// Check if the token is valid and not expired
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if exp, ok := claims["exp"].(float64); ok {
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				return nil, errors.New("Token expired")
			}
		}
		return token, nil
	}

	return nil, errors.New("Invalid token")
}

func CreteUser(User models.User) error{

	if len(User.FirstName) < 1{
		return errors.New("First name is required")
	}
	if len(User.LastName) < 1 {
		return errors.New("Last name is required")
	}
	if len(User.Email) < 1{
		return errors.New("Email is required")
	}
	if len(User.Password) < 1{
		return errors.New("Phone number is required")
	}

	userCollection := database.GetUserCollection()

	filter := bson.M{"email": User.Email}
	var existingUser models.User

	err := userCollection.FindOne(context.TODO(), filter).Decode(&existingUser)
	if err != nil && err != mongo.ErrNoDocuments{
		return err
	}

	if existingUser.Email != ""{
		return errors.New("User already exists")
	}

	for{
		User.Key = generateRandomString(10)

		User.Upi = generateUpi(10)

		keyFilter := bson.M{"key": User.Key}
		keyExists := userCollection.FindOne(context.TODO(), keyFilter).Err() == nil

		upiFilter := bson.M{"upi":User.Upi}
		upiExists := userCollection.FindOne(context.TODO(), upiFilter).Err() == nil

		if !keyExists && !upiExists{
			break
		}
	}

	User.Balance = 0

	hashedPassword, err := hashPassword(User.Password)
	if err != nil{
		return err
	}
	User.Password = hashedPassword

	_, err = userCollection.InsertOne(context.TODO(), User)
	return err
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodPost{
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var User models.User

	err := json.NewDecoder(r.Body).Decode(&User)
	if err != nil{
		http.Error(w, "Error decoding Body", http.StatusBadRequest)
		return
	}

	err = CreteUser(User)
	if err != nil{
		http.Error(w, err.Error(),  http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User Created Successfully"))
}

func LoginUser(email, password string)(string, error){
	userCollection := database.GetUserCollection()

	filter := bson.M{"email": email}

	var user models.User

	err := userCollection.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil{
		if err == mongo.ErrNoDocuments{
			return "", errors.New("User Not Found")
		}
		return "", err
	}

	if !verifyPassword(user.Password, password){
		return "", errors.New("Invalid Password")
	}

	token, err := generateToken(email)
	if err != nil{
		panic(err)
	}

	return token, nil
}

func LoginUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&loginRequest)
	if err != nil {
		http.Error(w, "Error decoding Body", http.StatusBadRequest)
		return
	}

	token, err := LoginUser(loginRequest.Email, loginRequest.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Login Successful"))
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request){
	if r.Method  != http.MethodGet{
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := validateToken(r)
	if err != nil{
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	userCollection := database.GetUserCollection()

	cursor, err := userCollection.Find(context.TODO(), bson.M{})
	if err != nil{
		http.Error(w, "Error retrieving User info", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var users []models.UserResponse

	for cursor.Next(context.Background()){
		var user models.User
		if err := cursor.Decode(&user); err != nil{
			http.Error(w, "Error decoding User info", http.StatusInternalServerError)
			return
		}
		userResponse := models.UserResponse{
			FirstName: user.FirstName,
            LastName:  user.LastName,
            Email:     user.Email,
            Upi:       user.Upi,
		}
		users = append(users, userResponse)
	}

	userJSON, err := json.Marshal(users)
	if err != nil{
		http.Error(w, "Error marshalling User info", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(userJSON)
}