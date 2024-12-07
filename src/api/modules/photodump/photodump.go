package photodump

import (
	"context"
	"errors"
	"home_api/src/database"
	"home_api/src/responses"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kolesa-team/goexiv"
)

// ------------------- Types -------------------

// Tags Type alias for the Tags "enum"
type Tags string

const (
	Sparkly   Tags = "sparkly"
	Christmas Tags = "christmas"
)

// Photo Struct for a photo
type Photo struct {
	ID          string    `json:"id"`
	File        url.URL   `json:"file"`
	Description string    `json:"description,omitempty"`
	Resolution  string    `json:"resolution"`
	TakenAt     time.Time `json:"taken_at,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
	ModifiedAt  time.Time `json:"modified_at"`
	People      []string  `json:"people,omitempty"`
	Tags        []Tags    `json:"tags,omitempty"`
}

// TagsString Converts the tags to strings, because type safety
func (p *Photo) TagsString() []string {
	var tags []string
	for _, tag := range p.Tags {
		tags = append(tags, string(tag))
	}
	return tags
}

// Unrwap Unwraps the Photo struct into an array of fields
func (p *Photo) Unwrap() []any {
	return []any{p.ID, p.File, p.Description, p.Resolution, p.TakenAt,
		p.UploadedAt, p.ModifiedAt, p.People, p.Tags}
}

// ------------------- Store -------------------

// PhotoStore Interface for the photo store
type PhotoStore interface {
	GetPhoto(id string) (*Photo, error)
	CreatePhoto(photo *Photo) error
	UpdatePhoto(photo *Photo) error
	DeletePhoto(id string) error
}

// store Private implementation of PhotoStore
type store struct {
	db *pgxpool.Pool
}

// NewStore Creates a new PhotoStore
func NewStore(db *pgxpool.Pool) PhotoStore {
	return &store{db: db}
}

// GetPhoto Get the specified Photo from the database
func (s *store) GetPhoto(id string) (*Photo, error) {
	rows, _ := s.db.Query(context.Background(), "SELECT * FROM photos WHERE id = $1", id)
	photo, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Photo])
	if err != nil {
		return nil, err
	}
	return photo, err
}

const insertQuery string = `
INSERT INTO photos
(id, file, description, resolution, taken_at, uploaded_at, modified_at, people, tags)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

// CreatePhoto Create a Photo entry in the database
func (s *store) CreatePhoto(p *Photo) error {
	_, err := s.db.Exec(context.Background(), insertQuery, p.Unwrap()...)
	if err != nil {
		return err
	}
	return nil
}

const updateQuery = `
UPDATE photos SET
file = $2, description = $3, resolution = $4, taken_at = $5,
uploaded_at = $6, modified_at = $7, people = $8, tags = $9)
WHERE id = $1`

// UpdatePhoto Update a Photo in the database
func (s *store) UpdatePhoto(p *Photo) error {
	_, err := s.db.Exec(context.Background(), updateQuery, p.Unwrap()...)
	if err != nil {
		return err
	}
	return nil
}

// DeletePhoto Delete a Photo in the database
func (s *store) DeletePhoto(id string) error {
	_, err := s.db.Query(context.Background(),
		"DELETE FROM PICTURES WHERE id = $1", id)
	if err != nil {
		// TODO: Upstream Differentiate between Server and Client Errors
		return err
	}
	return nil
}

// ------------------- Service -------------------

// PhotoService - Interface for the photo service
type PhotoService interface {
	GetPhoto(id string) (*Photo, error)
	UploadPhoto(photo *Photo) error
	EditPhoto(photo *Photo) error
	SafeDeletePhoto(id string, conirm string) error
}

// service Private PhotoService implementation
type service struct {
	PhotoStore
}

// NewService Creates a new PhotoService
func NewService(ps PhotoStore) PhotoService {
	return &service{ps}
}

// UploadPhoto Upload a new Photo
func (s *service) UploadPhoto(photo *Photo) error {
	return s.CreatePhoto(photo)
}

// EditPhoto Edit a Photo in the database
func (s *service) EditPhoto(photo *Photo) error {
	return s.UpdatePhoto(photo)
}

func (s *service) SafeDeletePhoto(id string, confirm string) error {
	// TODO: Check the photo's hash against the confirm string
	return nil
}

// ------------------- Handlers -------------------

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

// OldUploadPhoto - uploads a photo to the server
func OldUploadPhoto(w http.ResponseWriter, r *http.Request) {
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

	photo := &Photo{ID: id}

	file, header, err := r.FormFile("photo")
	if err == http.ErrMissingFile {
		log.Println("File not found")
		return nil, errors.New("file not found"), http.StatusBadRequest
	} else {
		status, err := AnalyzePhoto(file, header, photo)
		if err != nil {
			return nil, err, status
		}
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Println("Could not close file", err)
		}
	}(file)

	if desc := r.Form.Get("description"); desc != "" {
		photo.Description = desc
	}
	if people := r.Form.Get("people"); people != "" {
		people := strings.Split(people, ",")
		for _, name := range people {
			photo.People = append(photo.People, name)
		}
	}
	if tags := r.Form.Get("tags"); tags != "" {
		tags := strings.Split(tags, ",")
		for _, tag := range tags {
			photo.Tags = append(photo.Tags, Tags(tag))
		}
	}
	return photo, nil, http.StatusNoContent
}

// UploadPhoto - Upload a new photo
func UploadPhoto(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var photo *Photo
		var err error
		var code int

		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
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
		err = s.EditPhoto(&photo)
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
		// TODO: USE SAFE METHOD
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
		// var amount int
		// var err error
		// strAmount := r.URL.Query().Get("amount")
		// if strAmount == "" {
		// 	amount = 12
		// } else {
		// 	amount, err = strconv.Atoi(strAmount)
		// 	if err != nil {
		// 		log.Println("Invalid amount", err)
		// 		responses.BadRequest(w, r, "Invalid amount")
		// 		return
		// 	}
		// }
		// var cursor int
		// strCursor := r.URL.Query().Get("cursor")
		// if strCursor == "" {
		// 	cursor = 0
		// } else {
		// 	cursor, err = strconv.Atoi(strCursor)
		// 	if err != nil {
		// 		log.Println("Invalid cursor", err)
		// 		responses.BadRequest(w, r, "Invalid cursor")
		// 		return
		// 	}
		// }
		// var photos []Photo
		// if cursor >= len(s.photos) {
		// 	responses.BadRequest(w, r, "Invalid cursor")
		// 	return
		// }
		// for i := cursor; i < cursor+amount; i++ {
		// 	if i >= len(s.photos) {
		// 		break
		// 	}
		// 	photos = append(photos, s.photos[i])
		// }
		if r.Header.Get("Content-Type") == "" {
			// responses.SendComponent(w, r, PhotoCards(photos))
		} else {
			responses.StructOK(w, r, photos)
		}
	}
}
