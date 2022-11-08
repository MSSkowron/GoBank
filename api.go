package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type APIServer struct {
	listenAddr string
	storage    Storage
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

	router.HandleFunc("/account/{id}", makeHTTPHandleFunc(s.handleGetAccountByID))

	log.Println("JSON API server running on port: ", s.listenAddr)

	if err := http.ListenAndServe(s.listenAddr, router); err != nil {
		panic("Error while running APIServer: " + err.Error())
	}
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}
	if r.Method == "POST" {
		return s.handleCreateAccount(w, r)
	}
	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}

	return fmt.Errorf("Method %s is not supported by the JSON APIServer", r.Method)
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

	return WriteJSON(w, http.StatusOK, account)
}

// GET /account/{id}
func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	id := mux.Vars(r)["id"]

	return WriteJSON(w, http.StatusOK, NewAccount("firstname"+id, "lastname"))
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error string
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		}
	}
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
