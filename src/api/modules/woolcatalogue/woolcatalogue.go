package woolcatalogue

import (
	"errors"
	"home_api/src/responses"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/goccy/go-json"
)

// ------------------- Types -------------------

// Wool - Struct for wool
type Wool struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Brand       string `json:"brand,omitempty"`
	Weight      string `json:"weight,omitempty"`
	Length      string `json:"length,omitempty"`
	NeedleSize  string `json:"needle_size,omitempty"`
	Price       string `json:"price,omitempty"`
	Colour      string `json:"colour,omitempty"`
	Composition string `json:"composition,omitempty"`
}

// ------------------- Store -------------------

// WoolStore - Interface for the wool store
type WoolStore interface {
	GetWool(id int) (*Wool, error)
	CreateWool(wool *Wool) error
	UpdateWool(wool *Wool) error
	DeleteWool(id int) error
}

type store struct {
	wools []Wool
}

func Load() (*store, error) {
	// Hardcoded filename for now
	filename := "./data/woolcatalogue.json"
	wools := []Wool{}
	err := json.Unmarshal([]byte(filename), &wools)
	if err != nil {
		return nil, err
	}
	return &store{
		wools: wools,
	}, nil
}

func (s *store) Save() error {
	// Hardcoded filename for now
	filename := "./data/woolcatalogue.json"
	data, err := json.Marshal(s.wools)
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (s *store) GetWool(id int) (*Wool, error) {
	for _, wool := range s.wools {
		if wool.ID == id {
			return &wool, nil
		}
	}
	return nil, errors.New("wool not found")
}

func (s *store) CreateWool(wool *Wool) error {
	s.wools = append(s.wools, *wool)
	return s.Save()
}

func (s *store) UpdateWool(wool *Wool) error {
	for i, w := range s.wools {
		if w.ID == wool.ID {
			s.wools[i] = *wool
			return s.Save()
		}
	}
	return errors.New("wool not found")
}

func (s *store) DeleteWool(id int) error {
	for i, wool := range s.wools {
		if wool.ID == id {
			s.wools = append(s.wools[:i], s.wools[i+1:]...)
			return s.Save()
		}
	}
	return errors.New("wool not found")
}

// ------------------- Routes -------------------

// ApplyRoutes - Apply the routes to the API server
func ApplyRoutes(mux *http.ServeMux) *http.ServeMux {
	store, err := Load()
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /api/v1/woolcatalogue/wool", GetWool(store))
	mux.Handle("POST /api/v1/woolcatalogue/wool", CreateWool(store))
	mux.Handle("PUT /api/v1/woolcatalogue/wool", UpdateWool(store))
	mux.Handle("DELETE /api/v1/woolcatalogue/wool", DeleteWool(store))
	return mux
}

// CreateWool - Create a new wool
func CreateWool(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wool := Wool{}
		err := json.NewDecoder(r.Body).Decode(&wool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = s.CreateWool(&wool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// GetWool - Get a wool
func GetWool(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			responses.NotFound(w, r, "ID not found")
			return
		}
		intID, err := strconv.Atoi(id)
		if err != nil {
			log.Println("Could not convert ID to int", err)
			responses.NotFound(w, r, "Could not convert ID to int")
			return
		}
		wool, err := s.GetWool(intID)
		if err != nil {
			log.Println("Wool not found", err)
			responses.NotFound(w, r, "Wool not found")
			return
		}
		responses.StructOK(w, r, wool)
	}
}

// UpdateWool - Update a wool
func UpdateWool(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wool := Wool{}
		err := json.NewDecoder(r.Body).Decode(&wool)
		if err != nil {
			log.Println("Could not decode wool", err)
			responses.BadRequest(w, r, "Could not decode wool")
			return
		}
		err = s.UpdateWool(&wool)
		if err != nil {
			log.Println("Could not update wool", err)
			responses.BadRequest(w, r, "Could not update wool")
			return
		}
		responses.Success(w, r, "Wool updated successfully")
	}
}

// DeleteWool - Delete a wool
func DeleteWool(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the ID from the URL
		id := r.URL.Query().Get("id")
		if id == "" {
			responses.NotFound(w, r, "ID not found")
			return
		}
		intID, err := strconv.Atoi(id)
		if err != nil {
			log.Println("Could not convert ID to int", err)
			responses.NotFound(w, r, "Could not convert ID to int")
			return
		}
		err = s.DeleteWool(intID)
		if err != nil {
			log.Println("Could not delete wool", err)
			responses.NotFound(w, r, "Could not delete wool")
			return
		}
		responses.NoContent(w, r)
	}
}
