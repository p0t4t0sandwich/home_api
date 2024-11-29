package photodump

import (
	"errors"
	"home_api/src/database"
	"home_api/src/responses"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/kolesa-team/goexiv"
)

// ------------------- Types -------------------

type Tags string

const (
	Sparkly   Tags = "sparkly"
	Christmas Tags = "christmas"
)

type Person struct {
	Name string `json:"name"`
}

// File

// Photo - Struct for a photo
type Photo struct {
	ID          string    `json:"id"`
	File        url.URL   `json:"file"`
	Description string    `json:"description,omitempty"`
	Resolution  string    `json:"resolution"`
	TakenAt     time.Time `json:"taken_at,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
	ModifiedAt  time.Time `json:"modified_at"`
	People      []Person  `json:"people,omitempty"`
	Tags        []Tags    `json:"tags,omitempty"`
}

func (p Photo) TagsString() []string {
	var tags []string
	for _, tag := range p.Tags {
		tags = append(tags, string(tag))
	}
	return tags
}

func (p Photo) PeopleString() []string {
	var names []string
	for _, person := range p.People {
		names = append(names, string(person.Name))
	}
	return names
}

// ------------------- Store -------------------

// PhotoStore - Interface for the photo store
type PhotoStore interface {
	GetPhoto(id int) (*Photo, error)
	CreatePhoto(photo *Photo) error
	UpdatePhoto(photo *Photo) error
	DeletePhoto(id int) error
}

type store struct {
	photos []Photo
}

func Load() (*store, error) {
	// Hardcoded filename for now
	filename := "./data/photostore.json"
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var photos []Photo
	err = json.Unmarshal(file, &photos)
	if err != nil {
		return nil, err
	}
	return &store{
		photos: photos,
	}, nil
}

func (s *store) save() error {
	// Hardcoded filename for now
	filename := "./data/photostore.json"
	data, err := json.Marshal(s.photos)
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (s *store) GetPhoto(id string) (*Photo, error) {
	for _, photo := range s.photos {
		if photo.ID == id {
			return &photo, nil
		}
	}
	return nil, errors.New("photo not found")
}

func (s *store) CreatePhoto(photo *Photo) error {
	s.photos = append(s.photos, *photo)
	return s.save()
}

func (s *store) UpdatePhoto(photo *Photo) error {
	for i, w := range s.photos {
		if w.ID == photo.ID {
			s.photos[i] = *photo
			return s.save()
		}
	}
	return errors.New("photo not found")
}

func (s *store) DeletePhoto(id string) error {
	for i, photo := range s.photos {
		if photo.ID == id {
			s.photos = append(s.photos[:i], s.photos[i+1:]...)
			return s.save()
		}
	}
	return errors.New("photo not found")
}

// ------------------- Functions -------------------

// AnalyzePhoto - analyze a photo sent via multipartform
func AnalyzePhoto(file multipart.File, header *multipart.FileHeader, photo *Photo) (int, error) {
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Println("Could not read file", err)
		return http.StatusInternalServerError, err
	}

	fileType := http.DetectContentType(fileBytes)
	if fileType[:6] != "image/" {
		log.Println("File is not an image")
		return http.StatusBadRequest, errors.New("file is not an image")
	}

	var ext string
	switch fileType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	default:
		log.Println("Unsupported image type: " + fileType)
		return http.StatusBadRequest, errors.New("unsupported image type: " + fileType)
	}

	id, err := database.GenSnowflake()
	if err != nil {
		log.Println("Could not generate snowflake id", err)
		return http.StatusInternalServerError, errors.New("could not generate snowflake id")
	}

	// TODO: Convert to DB and S3
	fileName := "./data/photos/" + id + ext
	err = os.WriteFile(fileName, fileBytes, 0644)
	if err != nil {
		log.Println("Could not save file", err)
		return http.StatusInternalServerError, errors.New("could not save file")
	}

	return http.StatusNoContent, nil
}

// ------------------- API Routes -------------------

// UploadPhoto - uploads a photo to the server
func UploadPhoto(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		log.Println("Could not parse form", err)
		responses.BadRequest(w, r, "Could not parse form")
		return
	}

	file, _, err := r.FormFile("photo")

	if err != nil {
		log.Println("Could not get file from form", err)
		responses.BadRequest(w, r, "Could not get file from form")
		return
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Println("Could not close file", err)
		}
	}(file)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Println("Could not read file", err)
		responses.InternalServerError(w, r, "Could not read file")
		return
	}

	fileType := http.DetectContentType(fileBytes)
	if fileType[:6] != "image/" {
		log.Println("File is not an image")
		responses.BadRequest(w, r, "File is not an image")
		return
	}

	var ext string
	switch fileType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	default:
		log.Println("Unsupported image type: " + fileType)
		responses.BadRequest(w, r, "Unsupported image type")
		return
	}

	id, err := database.GenSnowflake()
	if err != nil {
		log.Println("Could not generate snowflake id", err)
		responses.InternalServerError(w, r, "Could not generate snowflake id")
		return
	}

	// TODO: Convert to DB and S3
	fileName := "./data/photos/" + id + ext
	err = os.WriteFile(fileName, fileBytes, 0644)
	if err != nil {
		log.Println("Could not save file", err)
		responses.InternalServerError(w, r, "Could not save file")
		return
	}

	PrintMeta(fileName)

	responses.NoContent(w)
}

func PrintMeta(fileName string) {
	img, err := goexiv.Open(fileName)
	if err != nil {
		log.Println("Could not read file from disk", err)
	}

	// Read metadata
	err = img.ReadMetadata()
	if err != nil {
		log.Println("Could not read image metadata", err)
	}

	err = img.ReadMetadata()
	if err != nil {
		return
	}
	// map[string]string
	exif := img.GetExifData().AllTags()
	log.Println(exif)

	// map[string]string
	iptc := img.GetIptcData().AllTags()
	log.Println(iptc)
}

// CreatePhotoFromFormData - Create a new photo from form data
func CreatePhotoFromFormData(r *http.Request) (*Photo, error, int) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		log.Println("Could not parse form", err)
		return nil, err, http.StatusBadRequest
	}
	id, err := database.GenSnowflake()
	if err != nil {
		log.Println("Could not generate ID", err)
		return nil, err, http.StatusInternalServerError
	}

	photo := Photo{ID: id}

	file, header, err := r.FormFile("photo")
	if err == http.ErrMissingFile {
		log.Println("File not found")
		return nil, errors.New("file not found"), http.StatusBadRequest
	} else {
		status, err := AnalyzePhoto(file, header, &photo)
		if err != nil {
			return nil, err, status
		}
	}
	if desc := r.Form.Get("description"); desc != "" {
		photo.Description = desc
	}
	if people := r.Form.Get("people"); people != "" {
		people := strings.Split(people, ",")
		for _, name := range people {
			photo.People = append(photo.People, Person{name})
		}
	}
	if tags := r.Form.Get("tags"); tags != "" {
		tags := strings.Split(tags, ",")
		for _, tag := range tags {
			photo.Tags = append(photo.Tags, Tags(tag))
		}
	}
	return &photo, nil, http.StatusNoContent
}

// CreatePhoto - Create a new photo
func CreatePhoto(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var photo *Photo
		var err error
		var code int
		if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			photo, err, code = CreatePhotoFromFormData(r)
		} else {
			// photo, err, code = CreatePhotoFromJSON(r)
		}
		if err != nil {
			switch code {
			case http.StatusBadRequest:
				responses.BadRequest(w, r, "Could not create photo")
				return
			case http.StatusInternalServerError:
				responses.InternalServerError(w, r, "Could not create photo")
				return
			}
		}
		err = s.CreatePhoto(photo)
		if err != nil {
			log.Println("Could not create photo", err)
			responses.InternalServerError(w, r, "Could not create photo")
			return
		}
		log.Println("Photo", photo.ID, "created successfully")
		responses.StructCreated(w, r, "Photo created successfully")
	}
}

// GetPhoto - Get a photo
func GetPhoto(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			log.Println("ID not found")
			responses.NotFound(w, r, "ID not found")
			return
		}
		photo, err := s.GetPhoto(id)
		if err != nil {
			log.Println("Photo not found", err)
			responses.NotFound(w, r, "Photo not found")
			return
		}
		log.Println("Photo", photo.ID, "found")
		responses.StructOK(w, r, photo)
	}
}

// UpdatePhoto - Update a photo
func UpdatePhoto(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		photo := Photo{}
		err := json.NewDecoder(r.Body).Decode(&photo)
		if err != nil {
			log.Println("Could not decode photo", err)
			responses.BadRequest(w, r, "Could not decode photo")
			return
		}
		err = s.UpdatePhoto(&photo)
		if err != nil {
			log.Println("Could not update photo", err)
			responses.BadRequest(w, r, "Could not update photo")
			return
		}
		log.Println("Photo", photo.ID, "updated successfully")
		responses.Success(w, r, "Photo updated successfully")
	}
}

// DeletePhoto - Delete a photo
func DeletePhoto(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the ID from the URL
		id := r.URL.Query().Get("id")
		if id == "" {
			responses.NotFound(w, r, "ID not found")
			return
		}
		err := s.DeletePhoto(id)
		if err != nil {
			log.Println("Could not delete photo", err)
			responses.NotFound(w, r, "Could not delete photo")
			return
		}
		log.Println("Photo", id, "deleted successfully")
		responses.NoContent(w)
	}
}

// GetPhotos - Get a list of photos
func GetPhotos(s *store) http.HandlerFunc {
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
		var photos []Photo
		if cursor >= len(s.photos) {
			responses.BadRequest(w, r, "Invalid cursor")
			return
		}
		for i := cursor; i < cursor+amount; i++ {
			if i >= len(s.photos) {
				break
			}
			photos = append(photos, s.photos[i])
		}
		if r.Header.Get("Content-Type") == "" {
			// responses.SendComponent(w, r, PhotoCards(photos))
		} else {
			responses.StructOK(w, r, photos)
		}
	}
}
