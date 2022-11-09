package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccounts() ([]*Account, error)
	GetAccountByID(int) (*Account, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	connStr := "user=gobank-postgres-user dbname=postgres password=gobank-postgres-password sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	postgresStore := &PostgresStore{db: db}
	if err := postgresStore.Init(); err != nil {
		return nil, err
	}

	return postgresStore, nil
}

func (s *PostgresStore) Init() error {
	return s.createAccountTable()
}

func (s *PostgresStore) createAccountTable() error {
	query := `create table if not exists account (
		id serial primary key,
		first_name varchar(50),
		last_name varchar(50),
		number serial,
		balance serial,
		created_at timestamp
	)`

	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateAccount(a *Account) error {
	query := `insert into account (first_name, last_name, number, balance, created_at) values ($1, $2, $3, $4, $5)`

	if _, err := s.db.Exec(query, a.FirstName, a.LastName, a.Number, a.Balance, a.CreatedAt); err != nil {
		return err
	}

	log.Println("[POSTGRES] Account correctly inserted to database.")

	return nil
}

func (s *PostgresStore) DeleteAccount(id int) error {
	query := `delete from account where id=$1`

	if _, err := s.db.Exec(query, id); err != nil {
		return err
	}

	log.Println("[POSTGRES] Account correctly deleted from database.")

	return nil
}

func (s *PostgresStore) UpdateAccount(a *Account) error {
	return nil
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	query := `select * from account`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	accounts := []*Account{}
	for rows.Next() {
		account := &Account{}
		if err := rows.Scan(&account.ID, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreatedAt); err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}

	log.Println("[POSTGRES] Accounts correctly pulled from database.")

	return accounts, nil
}

func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	query := `select * from account where id=$1`

	row := s.db.QueryRow(query, id)

	account := &Account{}
	if err := row.Scan(&account.ID, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreatedAt); err != nil {
		return nil, err
	}

	log.Println("[POSTGRES] Account correctly pulled from database.")

	return account, nil
}
