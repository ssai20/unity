package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"proGo/db"
	"proGo/internal/models"
	"time"

	_ "github.com/lib/pq"
)

func (s *Server) SetupRouters() *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("/members", s.handleMembers)
	router.HandleFunc("/", s.index)

	router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web"))))
	return router
}

type Server struct {
	db     *sql.DB
	router *http.ServeMux
}

func NewServer(config *db.Config) *Server {
	dB, err := initDB(config)
	if err != nil {
		log.Printf("FATAL: initDB failed: %v", err)
	}
	return &Server{
		db:     dB,
		router: http.NewServeMux(),
	}
}

const (
	appSchema = "app_schema_party"
)

func initDB(config *db.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=%s",
		config.DBUser, config.DBPassword,
		config.DBHost, config.DBPort,
		config.DBName, config.DBSSLMode,
		appSchema)
	log.Printf("Connecting to database with search_path=%s", appSchema)

	var db *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Attempt %d: sql.Open failed: %v", i+1, err)
			time.Sleep(3 * time.Second)
			continue
		}

		if err = db.Ping(); err != nil {
			log.Printf("Attempt %d: Ping failed: %v", i+1, err)
			db.Close()
			time.Sleep(3 * time.Second)
			continue
		}

		_, err = db.Exec(fmt.Sprintf("Create schema if not exists %s", appSchema))
		if err != nil {
			log.Printf("Warning: could not create schema: %v", err)
		}

		var hasUsage bool
		err = db.QueryRow(fmt.Sprintf(`SELECT has_schema_privilege(current_user, '%s', 'USAGE')`, appSchema)).Scan(&hasUsage)

		if err == nil && hasUsage {
			log.Printf("Successfully connected and have access to schema '%s'", appSchema)
			break
		}

		log.Printf("No access to schema '%s', retrying...", appSchema)
		db.Close()
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after retries: %w", err)
	}

	if err := createTable(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func createTable(db *sql.DB) error {

	regional := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.regional_committees (
			    id BIGSERIAL PRIMARY KEY,
			    name VARCHAR NOT NULL UNIQUE
			)
`, appSchema)

	if _, err := db.Exec(regional); err != nil {
		return fmt.Errorf("failed to create regional_committees: %w", err)
	}

	district := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.district_committees (
			    id BIGSERIAL PRIMARY KEY,
			    name VARCHAR NOT NULL,
			    regional_committee_id BIGINT NOT NULL
			                                                  REFERENCES %s.regional_committees(id) ON DELETE RESTRICT,
			    UNIQUE (regional_committee_id, name)
			)
`, appSchema, appSchema)

	if _, err := db.Exec(district); err != nil {
		return fmt.Errorf("failed to create district_committees: %w", err)
	}

	primary := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.primary_units (
			    id BIGSERIAL PRIMARY KEY,
			    name VARCHAR NOT NULL,
			    district_committee_id BIGINT NOT NULL
			                                            REFERENCES %s.district_committees(id) ON DELETE RESTRICT,
			    UNIQUE (district_committee_id, name)
			)
`, appSchema, appSchema)

	if _, err := db.Exec(primary); err != nil {
		return fmt.Errorf("failed to create primary_units: %w", err)
	}

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.members(
		    id BIGSERIAL PRIMARY KEY,
		    party_card_number VARCHAR NOT NULL UNIQUE,
		    first_name VARCHAR,
		    father_name VARCHAR,
		    last_name VARCHAR,
		    month INT,
		    year INT,
		    district_committee VARCHAR,
		    regional_committee VARCHAR,
		    birth_date DATE,
		    profession VARCHAR,
		    city VARCHAR,
		    street VARCHAR,
		    house INT,
		    building VARCHAR
		)        
`, appSchema)

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	log.Printf("Table %s.members,  %s.regional_committees, %s.district_committees, %s.primary_units are ready",
		appSchema, appSchema, appSchema, appSchema)
	return nil
}

func (s *Server) saveMember(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database connection not available", http.StatusBadRequest)
		return
	}

	var req models.PartyMember
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.members (party_card_number, first_name, father_name, last_name, month, year, district_committee,
		                        regional_committee, birth_date, profession, city, street, house, building) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`, appSchema)

	log.Printf("saveMember: %+v", req)
	if _, err := s.db.Exec(query, req.PartyCardNumber, req.Name.FirstName, req.Name.FatherName, req.Name.LastName, req.JoiningDate.Month, req.JoiningDate.Year,
		req.Committee.DistrictCommittee, req.Committee.RegionalCommittee, req.BirthDate, req.Profession, req.Address.City,
		req.Address.Street, req.Address.House, req.Address.Building); err != nil {
		log.Printf("saveMember: insert failed: %v", err)
		http.Error(w, "member can't save", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("good")
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}
func (s *Server) handleMembers(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.saveMember(w, r)
}
