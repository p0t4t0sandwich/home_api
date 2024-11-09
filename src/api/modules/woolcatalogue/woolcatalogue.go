package woolcatalogue

import (
	"errors"
	"home_api/src/database"
	"home_api/src/responses"
	"home_api/src/web/components"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/goccy/go-json"
)

// ------------------- Types -------------------

type Tags string

//goland:noinspection GoUnusedConst
const (
	Sparkly   Tags = "sparkly"
	Christmas Tags = "christmas"
)

// Wool - Struct for wool
type Wool struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Brand       string `json:"brand,omitempty"`
	Length      string `json:"length,omitempty"`
	Weight      string `json:"weight,omitempty"`
	Ply         int    `json:"ply,omitempty"`
	NeedleSize  string `json:"needle_size,omitempty"`
	Colour      string `json:"colour,omitempty"`
	Composition string `json:"composition,omitempty"`
	Quantity    int    `json:"quantity,omitempty"`
	Partial     int    `json:"partial,omitempty"`
	Tags        []Tags `json:"tags,omitempty"`
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

//goland:noinspection GoExportedFuncWithUnexportedType
func Load() (*store, error) {
	// Hardcoded filename for now
	filename := "./data/wool-catalogue.json"
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var wools []Wool
	err = json.Unmarshal(file, &wools)
	if err != nil {
		return nil, err
	}
	return &store{
		wools: wools,
	}, nil
}

func (s *store) save() error {
	// Hardcoded filename for now
	filename := "./data/wool-catalogue.json"
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

func (s *store) GetWool(id string) (*Wool, error) {
	for _, wool := range s.wools {
		if wool.ID == id {
			return &wool, nil
		}
	}
	return nil, errors.New("wool not found")
}

func (s *store) CreateWool(wool *Wool) error {
	s.wools = append(s.wools, *wool)
	return s.save()
}

func (s *store) UpdateWool(wool *Wool) error {
	for i, w := range s.wools {
		if w.ID == wool.ID {
			s.wools[i] = *wool
			return s.save()
		}
	}
	return errors.New("wool not found")
}

func (s *store) DeleteWool(id string) error {
	for i, wool := range s.wools {
		if wool.ID == id {
			s.wools = append(s.wools[:i], s.wools[i+1:]...)
			return s.save()
		}
	}
	return errors.New("wool not found")
}

// ------------------- API Routes -------------------

// CreateWool - Create a new wool
func CreateWool(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := database.GenSnowflake()
		if err != nil {
			responses.InternalServerError(w, r, "Could not generate ID")
			return
		}
		wool := Wool{}
		err = json.NewDecoder(r.Body).Decode(&wool)
		if err != nil {
			log.Println("Could not decode wool", err)
			responses.BadRequest(w, r, "Could not decode wool")
			return
		}
		wool.ID = id
		err = s.CreateWool(&wool)
		if err != nil {
			log.Println("Could not create wool", err)
			responses.BadRequest(w, r, "Could not create wool")
			return
		}
		log.Println("Wool", wool.ID, "created successfully")
		responses.StructCreated(w, r, wool)
	}
}

// GetWool - Get a wool
func GetWool(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			log.Println("ID not found")
			responses.NotFound(w, r, "ID not found")
			return
		}
		wool, err := s.GetWool(id)
		if err != nil {
			log.Println("Wool not found", err)
			responses.NotFound(w, r, "Wool not found")
			return
		}
		log.Println("Wool", wool.ID, "found")
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
		log.Println("Wool", wool.ID, "updated successfully")
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
		err := s.DeleteWool(id)
		if err != nil {
			log.Println("Could not delete wool", err)
			responses.NotFound(w, r, "Could not delete wool")
			return
		}
		log.Println("Wool", id, "deleted successfully")
		responses.NoContent(w, r)
	}
}

// GetWools - Get a list of wools
func GetWools(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strAmount := r.URL.Query().Get("amount")
		if strAmount == "" {
			strAmount = "10"
		}
		strCursor := r.URL.Query().Get("cursor")
		if strCursor == "" {
			strCursor = "0"
		}
		amount, err := strconv.Atoi(strAmount)
		if err != nil {
			responses.BadRequest(w, r, "Invalid amount")
			return
		}
		cursor, err := strconv.Atoi(strCursor)
		if err != nil {
			responses.BadRequest(w, r, "Invalid cursor")
			return
		}
		var wools []Wool
		if cursor >= len(s.wools) {
			responses.BadRequest(w, r, "Invalid cursor")
			return
		}
		for i := cursor; i < cursor+amount; i++ {
			if i >= len(s.wools) {
				break
			}
			wools = append(wools, s.wools[i])
		}
		responses.StructOK(w, r, wools)
	}
}

// ------------------- HTML Routes -------------------

// GetWoolsHTML - Get a list of wools as HTML
func GetWoolsHTML(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strAmount := r.URL.Query().Get("amount")
		if strAmount == "" {
			strAmount = "10"
		}
		strCursor := r.URL.Query().Get("cursor")
		if strCursor == "" {
			strCursor = "0"
		}
		amount, err := strconv.Atoi(strAmount)
		if err != nil {
			responses.BadRequest(w, r, "Invalid amount")
			return
		}
		cursor, err := strconv.Atoi(strCursor)
		if err != nil {
			responses.BadRequest(w, r, "Invalid cursor")
			return
		}
		var wools []Wool
		if cursor >= len(s.wools) {
			responses.BadRequest(w, r, "Invalid cursor")
			return
		}
		for i := cursor; i < cursor+amount; i++ {
			if i >= len(s.wools) {
				break
			}
			wools = append(wools, s.wools[i])
		}
		responses.SendComponent(w, r, components.Wools(wools))
	}
}
