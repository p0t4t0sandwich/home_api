package woolcatalogue

import (
	"errors"
	"home_api/src/database"
	"home_api/src/responses"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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
	Name        string `json:"name"`
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

func (w Wool) TagsString() []string {
	var tags []string
	for _, tag := range w.Tags {
		tags = append(tags, string(tag))
	}
	return tags
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

// CreateWoolFromFormData - Create a new wool from form data
func CreateWoolFromFormData(r *http.Request) (*Wool, error, int) {
	err := r.ParseForm()
	if err != nil {
		log.Println("Could not parse form", err)
		return nil, err, http.StatusBadRequest
	}
	id, err := database.GenSnowflake()
	if err != nil {
		log.Println("Could not generate ID", err)
		return nil, err, http.StatusInternalServerError
	}
	wool := Wool{ID: id}
	if name := r.Form.Get("name"); name != "" {
		wool.Name = name
	} else {
		log.Println("Name not found")
		return nil, errors.New("name not found"), http.StatusBadRequest
	}
	if brand := r.Form.Get("brand"); brand != "" {
		wool.Brand = brand
	}
	if length := r.Form.Get("length"); length != "" {
		wool.Length = length
	}
	if weight := r.Form.Get("weight"); weight != "" {
		wool.Weight = weight
	}
	if ply := r.Form.Get("ply"); ply != "" {
		plyInt, err := strconv.Atoi(ply)
		if err != nil {
			log.Println("Could not convert ply to int", err)
			return nil, err, http.StatusBadRequest
		}
		wool.Ply = plyInt
	}
	if needleSize := r.Form.Get("needle_size"); needleSize != "" {
		wool.NeedleSize = needleSize
	}
	if colour := r.Form.Get("colour"); colour != "" {
		wool.Colour = colour
	}
	if composition := r.Form.Get("composition"); composition != "" {
		wool.Composition = composition
	}
	if quantity := r.Form.Get("quantity"); quantity != "" {
		quantityInt, err := strconv.Atoi(quantity)
		if err != nil {
			log.Println("Could not convert quantity to int", err)
			return nil, err, http.StatusBadRequest
		}
		wool.Quantity = quantityInt
	}
	if partial := r.Form.Get("partial"); partial != "" {
		partialInt, err := strconv.Atoi(partial)
		if err != nil {
			log.Println("Could not convert partial to int", err)
			return nil, err, http.StatusBadRequest
		}
		wool.Partial = partialInt
	}
	if tags := r.Form.Get("tags"); tags != "" {
		tags := strings.Split(tags, ",")
		for _, tag := range tags {
			wool.Tags = append(wool.Tags, Tags(tag))
		}
	}
	return &wool, nil, http.StatusOK
}

// CreateWoolFromJSON - Create a new wool from JSON
func CreateWoolFromJSON(r *http.Request) (*Wool, error, int) {
	id, err := database.GenSnowflake()
	if err != nil {
		log.Println("Could not generate ID", err)
		return nil, err, http.StatusInternalServerError
	}
	wool := Wool{ID: id}
	err = json.NewDecoder(r.Body).Decode(&wool)
	if err != nil {
		log.Println("Could not decode wool", err)
		return nil, err, http.StatusBadRequest
	}
	return &wool, nil, http.StatusOK
}

// CreateWool - Create a new wool
func CreateWool(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var wool *Wool
		var err error
		var code int
		if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			wool, err, code = CreateWoolFromFormData(r)
		} else {
			wool, err, code = CreateWoolFromJSON(r)
		}
		if err != nil {
			switch code {
			case http.StatusBadRequest:
				responses.BadRequest(w, r, "Could not create wool")
				return
			case http.StatusInternalServerError:
				responses.InternalServerError(w, r, "Could not create wool")
				return
			}
		}
		err = s.CreateWool(wool)
		if err != nil {
			log.Println("Could not create wool", err)
			responses.InternalServerError(w, r, "Could not create wool")
			return
		}
		log.Println("Wool", wool.ID, "created successfully")
		responses.StructCreated(w, r, "Wool created successfully")
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
		responses.NoContent(w)
	}
}

// GetWools - Get a list of wools
func GetWools(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var amount int
		var err error
		strAmount := r.URL.Query().Get("amount")
		if strAmount == "" {
			amount = 12
		} else {
			amount, err = strconv.Atoi(strAmount)
			if err != nil {
				log.Println("Invalid amount", err)
				responses.BadRequest(w, r, "Invalid amount")
				return
			}
		}
		var cursor int
		strCursor := r.URL.Query().Get("cursor")
		if strCursor == "" {
			cursor = 0
		} else {
			cursor, err = strconv.Atoi(strCursor)
			if err != nil {
				log.Println("Invalid cursor", err)
				responses.BadRequest(w, r, "Invalid cursor")
				return
			}
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
		if r.Header.Get("Content-Type") == "" {
			responses.SendComponent(w, r, WoolCards(wools))
		} else {
			responses.StructOK(w, r, wools)
		}
	}
}
