package Controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"payment/database"
	models "payment/model"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/dgrijalva/jwt-go"
	"github.com/joho/godotenv"

	// "github.com/pelletier/go-toml/query"
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

var cld *cloudinary.Cloudinary

func init(){
	if err := godotenv.Load(); err != nil{
		panic("error in loading .env file")
		return
	}

	cld, _ = cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
}

func uploadImageToCloud(imagePath string) (string, error){
	resp,err := cld.Upload.Upload(context.TODO(), imagePath, uploader.UploadParams{Folder: "images"})
	if err != nil{
		return "", err
	}
	return resp.SecureURL, nil
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

	if User.TransactionID == nil{
		User.TransactionID = []string{}
	}

	hashedPassword, err := hashPassword(User.Password)
	if err != nil{
		return err
	}
	User.Password = hashedPassword

	_, err = userCollection.InsertOne(context.TODO(), User)
	return err
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }

    var User models.User

    err := json.NewDecoder(r.Body).Decode(&User)
    if err != nil {
        http.Error(w, "Error decoding Body", http.StatusBadRequest)
        return
    }

    // Debug the image URL
    if User.ImageURL != "" {
        // Convert path if needed
        imagePath := strings.ReplaceAll(User.ImageURL, "\\", "/")
        fmt.Printf("Image Path: %s\n", imagePath) // Debug statement

        imageURL, err := uploadImageToCloud(imagePath)
        if err != nil {
            http.Error(w, "Error uploading image", http.StatusInternalServerError)
            return
        }
        User.ImageURL = imageURL
        fmt.Printf("Uploaded Image URL: %s\n", User.ImageURL) // Debug statement
    } else {
        // Log or handle the case where the image URL is empty
        fmt.Println("Image URL is empty")
    }

    err = CreteUser(User)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    w.Write([]byte("User Created Successfully"))
}


func LoginUser(key string)(string, error){
	userCollection := database.GetUserCollection()

	filter := bson.M{"key": key}

	var user models.User
	err := userCollection.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil{
		if err == mongo.ErrNoDocuments{
			return "", errors.New("User not found")
		}
		return "", err
	}

	// filter := bson.M{"email": email}

	// var user models.User

	// err := userCollection.FindOne(context.TODO(), filter).Decode(&user)
	// if err != nil{
	// 	if err == mongo.ErrNoDocuments{
	// 		return "", errors.New("User Not Found")
	// 	}
	// 	return "", err
	// }

	// if !verifyPassword(user.Password, password){
	// 	return "", errors.New("Invalid Password")
	// }


	token, err := generateToken(key)
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

    // Retrieve 'key' from query parameters
    key := r.URL.Query().Get("key")

    // Optionally: Retrieve 'key' from request body (if needed)
    // var loginRequest struct {
    //     Key string `json:"key"`
    // }
    // err := json.NewDecoder(r.Body).Decode(&loginRequest)
    // if err != nil {
    //     http.Error(w, "Error decoding Body", http.StatusBadRequest)
    //     return
    // }
    // key = loginRequest.Key

    if key == "" {
        http.Error(w, "Key is required", http.StatusBadRequest)
        return
    }

    token, err := LoginUser(key)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }

    w.WriteHeader(http.StatusOK)
    response := map[string]string{"token": token}
    json.NewEncoder(w).Encode(response)
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
func UpdateKey(key, email, password string) (string, error) {
    userCollection := database.GetUserCollection()

    // Find the user
    filter := bson.M{"email": email}
    var user models.User
    
    err := userCollection.FindOne(context.Background(), filter).Decode(&user)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return "", errors.New("User not found")
        }
        return "", err
    }

    // Verify password
    if !verifyPassword(user.Password, password) {
        return "", errors.New("Invalid password")
    }

    // Update key
    updateFilter := bson.M{"email": email}
    update := bson.M{"$set": bson.M{"key": key}}
    
    result := userCollection.FindOneAndUpdate(context.Background(), updateFilter, update)

	log.Printf("Updating user with email: %s\n", email)
	log.Printf("New key: %s\n", key)


    // Check if the update was successful
    if result.Err() != nil {
        return "", result.Err()
    }

    return "Key Updated Successfully", nil
}

func UpdateKeyHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut && r.Method != http.MethodPatch {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    key := r.URL.Query().Get("key")

    if key == "" {
        http.Error(w, "Key is required", http.StatusBadRequest)
        return
    }

    var requestBody struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }

    err := json.NewDecoder(r.Body).Decode(&requestBody)
    if err != nil {
        http.Error(w, "Error parsing JSON", http.StatusBadRequest)
        return
    }

    message, err := UpdateKey(key, requestBody.Email, requestBody.Password)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte(message))
}
