package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	jwt "github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

type APIServer struct {
	listenAddr string
	storage    Storage
}

type APIError struct {
	Error string
}

type apiFunc func(http.ResponseWriter, *http.Request) error

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			if err := WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()}); err != nil {
				log.Printf("[API] Error while writing error to JSON: %s", err.Error())
			}
		}
	}
}

func NewAPIServer(listenAddr string, storage Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		storage:    storage,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleAccountByID), s.storage))
	router.HandleFunc("/transfer", makeHTTPHandleFunc(s.handleTransfer))

	log.Println("[API] Server running on port:", s.listenAddr[1:])

	if err := http.ListenAndServe(s.listenAddr, router); err != nil {
		log.Fatal("[API] Error while running APIServer: " + err.Error())
	}
}

// handler /account
func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}
	if r.Method == "POST" {
		return s.handleCreateAccount(w, r)
	}

	return fmt.Errorf("[API] Method %s is not supported by the API/account", r.Method)
}

// GET /account
func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.storage.GetAccounts()
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

// POST /account
func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountReq := CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(&createAccountReq); err != nil {
		return err
	}

	account := NewAccount(createAccountReq.FirstName, createAccountReq.LastName)
	if err := s.storage.CreateAccount(account); err != nil {
		return err
	}

	tokenString, err := createJWT(account)
	if err != nil {
		return err
	}

	fmt.Println("JWT TOKEN: ", tokenString)

	return WriteJSON(w, http.StatusOK, account)
}

// handler /account/{id}
func (s *APIServer) handleAccountByID(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccountByID(w, r)
	}
	if r.Method == "DELETE" {
		return s.handleDeleteAccountByID(w, r)
	}

	return fmt.Errorf("[API] Method %s is not supported by the API/account/{id}", r.Method)
}

// GET /account/{id}
func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	id, err := getIDFromRequest(r)
	if err != nil {
		return err
	}

	account, err := s.storage.GetAccountByID(id)
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, Account{
		ID:        account.ID,
		FirstName: account.FirstName,
		LastName:  account.LastName,
		Number:    account.Number,
		Balance:   account.Balance,
		CreatedAt: account.CreatedAt,
	})
}

// DELETE /account/{id}
func (s *APIServer) handleDeleteAccountByID(w http.ResponseWriter, r *http.Request) error {
	id, err := getIDFromRequest(r)
	if err != nil {
		return err
	}

	if err := s.storage.DeleteAccount(id); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, map[string]int{"deleted": id})
}

// POST /transfer
func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("[API] Method %s is not supported by the API/transfer", r.Method)
	}

	transferRequest := TransferRequest{}
	if err := json.NewDecoder(r.Body).Decode(&transferRequest); err != nil {
		return err
	}
	defer r.Body.Close()

	if err := s.storage.Transfer(transferRequest.AccountNumber, transferRequest.Amount); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, transferRequest)
}

func getIDFromRequest(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return -1, fmt.Errorf("Invalid account ID given %s", idStr)
	}

	return id, nil
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt":     15000,
		"accountNumber": account.Number,
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("calling JWT auth middleware")

		tokenString := r.Header.Get("x-jwt-token")

		token, err := validateJWT(tokenString)
		if err != nil {
			WriteJSON(w, http.StatusForbidden, APIError{Error: "permission denied"})
			return
		}
		if !token.Valid {
			WriteJSON(w, http.StatusForbidden, APIError{Error: "permission denied"})
			return
		}

		userID, err := getIDFromRequest(r)
		if err != nil {
			WriteJSON(w, http.StatusForbidden, APIError{Error: "permission denied"})
			return
		}

		account, err := s.GetAccountByID(userID)
		if err != nil {
			WriteJSON(w, http.StatusForbidden, APIError{Error: "permission denied"})
			return
		}

		if account.Number != int64(token.Claims.(jwt.MapClaims)["accountNumber"].(float64)) {
			WriteJSON(w, http.StatusForbidden, APIError{Error: "permission denied"})
			return
		}

		handlerFunc(w, r)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
}
