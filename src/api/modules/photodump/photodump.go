package photodump

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"home_api/src/database"
	"home_api/src/responses"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/corona10/goimagehash"
	"github.com/goccy/go-json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kolesa-team/goexiv"
	"github.com/minio/minio-go/v7"
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
	ID          string    `json:"id" db:"id"`
	File        string    `json:"file" db:"file"`
	Ext         string    `json:"ext" db:"ext"`
	Hash        string    `json:"hash" db:"hash"`
	PHash       []byte    `json:"phash" db:"phash"`
	Description string    `json:"description" db:"description"`
	Source      string    `json:"source" db:"source"`
	Subjects    []string  `json:"subjects" db:"subjects"`
	Tags        []Tags    `json:"tags" db:"tags"`
	Resolution  string    `json:"resolution" db:"resolution"`
	TakenAt     time.Time `json:"taken_at" db:"taken_at"`
	UploadedAt  time.Time `json:"uploaded_at" db:"uploaded_at"`
	ModifiedAt  time.Time `json:"modified_at" db:"modified_at"`
}

// GetImgData Get the image data from a file and add it to the photo
func (p *Photo) GetImgData(r io.Reader, bs []byte, contentType string) (int, error) {
	var err error
	var img image.Image
	var ext string
	switch contentType {
	case "image/jpeg":
		ext = "jpg"
		img, err = jpeg.Decode(r)
		break
	case "image/png":
		ext = "png"
		img, err = png.Decode(r)
		break
	case "image/gif":
		ext = "gif"
		img, err = gif.Decode(r)
		break
	case "image/webp":
		ext = "webp"
		img, err = webp.Decode(r)
		break
	default:
		log.Println("unsupported image type: " + contentType + ". ID: " + p.ID)
		return http.StatusBadRequest, errors.New("unsupported image type: " + contentType)
	}
	if err != nil {
		log.Println("error reading image. ID: "+p.ID, err)
		return http.StatusBadRequest, errors.New("error reading image")
	}
	p.Ext = ext

	ph, err := goimagehash.PerceptionHash(img)
	if err != nil {
		log.Println("error generating phash. ID: "+p.ID, err)
		return http.StatusBadRequest, errors.New("error generating phash")
	}
	iph := ph.GetHash()
	phash := make([]byte, 8)
	binary.LittleEndian.PutUint64(phash, iph)
	p.PHash = phash

	sha := sha256.Sum256(bs)
	p.Hash = hex.EncodeToString(sha[:])

	w := strconv.Itoa(img.Bounds().Dx())
	h := strconv.Itoa(img.Bounds().Dy())
	p.Resolution = w + "x" + h + "p"

	log.Println("photo data aquired. ID: " + p.ID)
	return http.StatusCreated, nil
}

// GetExivData Get Exiv2 data from a file and and add it to the photo
func (p *Photo) GetExivData(bs []byte) error {
	img, err := goexiv.OpenBytes(bs)
	if err == nil {
		err = img.ReadMetadata()
		if err == nil {
			// "yyyy:MM:dd HH:mm:ss"
			var dt string
			exif := img.GetExifData().AllTags()
			dt = exif["Exif.Image.DateTimeOriginal"]
			if dt == "" {
				dt = exif["Exif.Photo.DateTimeOriginal"]
			}
			if dt == "" {
				iptc := img.GetIptcData().AllTags()
				dt = iptc["Date Created"]
			}
			if dt != "" {
				dateStr := strings.Replace(dt, ":", "-", 2)
				t, err := time.Parse(time.DateTime, dateStr)
				if err == nil {
					p.TakenAt = t
				}
			}
		} else {
			log.Println("could not retrieve photo metadata. ID: "+p.ID, err)
		}
	} else {
		log.Println("could not retrieve photo metadata. ID: "+p.ID, err)
	}
	return nil
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
	return []any{p.ID, p.File, p.Ext, p.Hash, p.PHash,
		p.Description, p.Source, p.Subjects, p.Tags,
		p.Resolution, p.TakenAt, p.UploadedAt, p.ModifiedAt}
}

// ------------------- Store -------------------

// PhotoStore Interface for the photo store
type PhotoStore interface {
	GetPhotoById(id string) (*Photo, error)
	GetPhotoByHash(hash string) (*Photo, error)
	CountLikePhotos(phash []byte, hd int) (int, error)
	CreatePhoto(photo *Photo) error
	UpdatePhoto(photo *Photo) error
	DeletePhoto(id string) error
	UploadPhotoToS3(photo *Photo, r io.Reader, length int64, contentType string) error
	DeletePhotoFromS3(photo *Photo) error
}

// store Private implementation of PhotoStore
type store struct {
	db *pgxpool.Pool
	s3 *minio.Client
}

// NewStore Creates a new PhotoStore
func NewStore(db *pgxpool.Pool, s3 *minio.Client) PhotoStore {
	return &store{db, s3}
}

// GetPhotoById Get the specified Photo from the database
func (s *store) GetPhotoById(id string) (*Photo, error) {
	rows, _ := s.db.Query(context.Background(), "SELECT * FROM photos WHERE id = $1", id)
	photo, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Photo])
	if err != nil {
		return nil, err
	}
	if photo.Subjects == nil {
		photo.Subjects = make([]string, 0)
	}
	if photo.Tags == nil {
		photo.Tags = make([]Tags, 0)
	}
	return photo, err
}

// GetPhotoByHash Get the specified Photo from the database
func (s *store) GetPhotoByHash(hash string) (*Photo, error) {
	rows, _ := s.db.Query(context.Background(), "SELECT * FROM photos WHERE hash = $1", hash)
	photo, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Photo])
	if err != nil {
		return nil, err
	}
	if photo.Subjects == nil {
		photo.Subjects = make([]string, 0)
	}
	if photo.Tags == nil {
		photo.Tags = make([]Tags, 0)
	}
	return photo, err
}

const checkpHashQuery = `
SELECT COUNT(*) FROM photos
WHERE BIT_COUNT(xor_digests(phash, $1)) <= $2`

// CountLikePhotos Return the number of similar photos
func (s *store) CountLikePhotos(phash []byte, hd int) (int, error) {
	// TODO: Compare rotated hashes? (90deg, 180deg)
	var count int
	err := s.db.QueryRow(context.Background(), checkpHashQuery, phash, hd).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

const insertQuery string = `
INSERT INTO photos
(id, file, ext, hash, phash,
description, source, subjects, tags,
resolution, taken_at, uploaded_at, modified_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

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
file = $2, ext = $3, hash = $4, phash = $5,
description = $6, source = $7, subjects = $8, tags = $9,
resolution = $10, taken_at = $11, uploaded_at = $12, modified_at = $13)
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
		"DELETE FROM photos WHERE id = $1", id)
	if err != nil {
		return err
	}
	return nil
}

// UploadPhotoToS3 Upload a photo to S3
func (s *store) UploadPhotoToS3(photo *Photo, r io.Reader, length int64, contentType string) error {
	_, err := s.s3.PutObject(
		context.Background(), "photos", photo.ID+"."+photo.Ext, r, length,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return err
	}
	photo.File = database.S3_FILE_URI + "/photos/" + photo.ID + "." + photo.Ext
	return nil
}

// DeletePhotoFromS3 Delete a photo from S3
func (s *store) DeletePhotoFromS3(photo *Photo) error {
	err := s.s3.RemoveObject(
		context.Background(), "photos", photo.ID+"."+photo.Ext,
		minio.RemoveObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}

// ------------------- Service -------------------

// PhotoService Interface for the photo service
type PhotoService interface {
	GetPhotoById(id string) (*Photo, int, error)
	GetPhotoByHash(hash string) (*Photo, int, error)
	UploadPhoto(photo *Photo, file *os.File) (int, error)
	EditPhoto(photo *Photo) (int, error)
	SafeDeletePhoto(id string, confirm string) (int, error)
}

// service Private PhotoService implementation
type service struct {
	ps PhotoStore
}

// NewService Creates a new PhotoService
func NewService(ps PhotoStore) PhotoService {
	return &service{ps}
}

// GetPhotoById Get the specified Photo from the database
func (s *service) GetPhotoById(id string) (*Photo, int, error) {
	// TODO: Differentiate between Server and Client caused db Errors
	photo, err := s.ps.GetPhotoById(id)
	if err != nil {
		log.Println("could not get photo. ID: "+id, err)
		return nil, http.StatusNotFound, errors.New("photo does not exist")
	}
	log.Println("got photo by id. ID: " + photo.ID)
	return photo, http.StatusOK, nil
}

// GetPhotoByHash Get the specified Photo from the database
func (s *service) GetPhotoByHash(hash string) (*Photo, int, error) {
	// TODO: Differentiate between Server and Client caused db Errors
	photo, err := s.ps.GetPhotoByHash(hash)
	if err != nil {
		log.Println("could not get photo. Hash: "+hash, err)
		return nil, http.StatusNotFound, errors.New("photo does not exist")
	}
	log.Println("got photo by hash. ID: " + photo.ID)
	return photo, http.StatusOK, nil
}

// UploadPhoto Upload a new Photo
func (s *service) UploadPhoto(photo *Photo, file *os.File) (int, error) {
	// TODO: Differentiate between Server and Client caused db Errors
	id, err := database.GenSnowflake()
	if err != nil {
		log.Println("could not generate id", err)
		return http.StatusInternalServerError, errors.New("could not generate id")
	}
	photo.ID = id

	bs, err := io.ReadAll(file)
	if err != nil && err != io.EOF {
		log.Println("could not read file contents. ID: "+photo.ID, err)
		return http.StatusBadRequest, errors.New("could not read file contents")
	}
	if len(bs) == 0 {
		log.Println("file is empty. ID: "+photo.ID, err)
		return http.StatusBadRequest, errors.New("file is empty")
	}

	info, err := file.Stat()
	if err != nil {
		log.Println("could not read file info. ID: "+photo.ID, err)
		return http.StatusBadRequest, errors.New("could not read file info")
	}
	photo.TakenAt = info.ModTime()
	photo.ModifiedAt = info.ModTime()

	contentType := http.DetectContentType(bs)
	if contentType[:6] != "image/" {
		log.Println("file is not an image: " + contentType + ". ID: " + photo.ID)
		return http.StatusBadRequest, errors.New("file is not an image: " + contentType)
	}

	nbs := make([]byte, len(bs))
	copy(nbs, bs)
	status, err := photo.GetImgData(bytes.NewBuffer(nbs), bs, contentType)
	if err != nil {
		return http.StatusBadRequest, err
	}

	hd := 0
	limit := 0
	count, err := s.ps.CountLikePhotos(photo.PHash, hd)
	if err != nil {
		log.Println("failed to count like photos. ID: "+photo.ID, err)
		return http.StatusInternalServerError, errors.New("failed to count like photos")
	}
	if count > limit {
		log.Println("duplicate image. ID: " + photo.ID)
		return http.StatusBadRequest, errors.New("duplicate image")
	}

	err = photo.GetExivData(bs)
	if err != nil {
		log.Println("Exiv analysis failed. ID: "+photo.ID, err)
	}

	nbs = make([]byte, len(bs))
	copy(nbs, bs)
	err = s.ps.UploadPhotoToS3(photo, bytes.NewBuffer(nbs), info.Size(), contentType)
	if err != nil {
		log.Println("could not upload photo to S3. ID: "+photo.ID, err)
		return http.StatusInternalServerError, errors.New("could not upload photo to S3")
	}

	err = s.ps.CreatePhoto(photo)
	if err != nil {
		log.Println("could not upload photo. ID: "+photo.ID, err)
		err = s.ps.DeletePhotoFromS3(photo)
		if err != nil {
			log.Println("could not remove photo from S3 after failing to upload photo: ID: "+photo.ID, err)
			return http.StatusInternalServerError, errors.New("could not remove photo from S3 after failing to upload photo")
		}
		log.Println("removed in-progress upload from S3. ID: " + photo.ID)
		return http.StatusInternalServerError, errors.New("could not upload photo")
	}
	log.Println("photo uploaded successfully. ID: " + photo.ID)
	return status, nil
}

// EditPhoto Edit a Photo in the database
func (s *service) EditPhoto(photo *Photo) (int, error) {
	// TODO: Differentiate between Server and Client caused db Errors
	photo, status, err := s.GetPhotoById(photo.ID)
	if err != nil {
		return status, err
	}

	photo.ModifiedAt = time.Now()
	err = s.ps.UpdatePhoto(photo)
	if err != nil {
		log.Println("could not update photo. ID: "+photo.ID, err)
		return http.StatusInternalServerError, errors.New("could not update photo")
	}
	log.Println("edited photo. ID: " + photo.ID)
	return http.StatusNoContent, nil
}

// SafeDeletePhoto Delete a photo if the confirmation string matches the hash
func (s *service) SafeDeletePhoto(id string, confirm string) (int, error) {
	// TODO: Differentiate between Server and Client caused db Errors
	photo, status, err := s.GetPhotoById(id)
	if err != nil {
		return status, err
	}
	if photo.Hash != confirm {
		return http.StatusBadRequest, errors.New("confirmation hash does not match photo hash")
	}
	err = s.ps.DeletePhotoFromS3(photo)
	if err != nil {
		log.Println("could not remove photo from S3", err)
		return http.StatusInternalServerError, errors.New("could not remove photo from S3")
	}
	err = s.ps.DeletePhoto(id)
	if err != nil {
		log.Println("could not delete photo", err)
		return http.StatusInternalServerError, errors.New("could not delete photo")
	}
	return http.StatusNoContent, nil
}

// ------------------- Functions -------------------

// CreatePhotoFromFormData - Create a new photo from form data
func CreatePhotoFromFormData(s PhotoService, r *http.Request) (*Photo, int, error) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		log.Println("could not parse form", err)
		return nil, http.StatusBadRequest, errors.New("could not parse form")
	}

	photo := &Photo{}
	if desc := r.Form.Get("description"); desc != "" {
		photo.Description = desc
	}
	if subjects := r.Form.Get("subjects"); subjects != "" {
		subjects := strings.Split(subjects, ",")
		for _, subject := range subjects {
			photo.Subjects = append(photo.Subjects, subject)
		}
	}
	if tags := r.Form.Get("tags"); tags != "" {
		tags := strings.Split(tags, ",")
		for _, tag := range tags {
			photo.Tags = append(photo.Tags, Tags(tag))
		}
	}

	mFile, _, err := r.FormFile("photo")
	if err == http.ErrMissingFile {
		log.Println("file not uploaded")
		return nil, http.StatusBadRequest, errors.New("file not uploaded")
	} else {
		file, ok := mFile.(*os.File)
		if !ok {
			return nil, http.StatusBadRequest, errors.New("invalid file")
		}
		status, err := s.UploadPhoto(photo, file)
		if err != nil {
			return nil, status, err
		}
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Println("could not close file", err)
		}
	}(mFile)

	return photo, http.StatusCreated, nil
}

// ------------------- Handlers -------------------

// UploadPhoto Upload a new photo
func UploadPhoto(s PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var photo *Photo
		var err error
		var status int

		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			photo, status, err = CreatePhotoFromFormData(s, r)
		} else {
			// photo, err, code = CreatePhotoFromJSON(r)
		}
		if err != nil {
			responses.SwitchCase(w, r, status, err.Error())
			return
		}
		log.Println("photo", photo.ID, "created successfully")
		responses.StructCreated(w, r, &photo)
	}
}

// GetPhoto Get a photo
func GetPhoto(s PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			responses.BadRequest(w, r, "no ID in the query")
			return
		}
		photo, status, err := s.GetPhotoById(id)
		if err != nil {
			responses.SwitchCase(w, r, status, err.Error())
			return
		}
		log.Println("photo", photo.ID, "found")
		responses.StructOK(w, r, photo)
	}
}

// UpdatePhoto Update a photo
func UpdatePhoto(s PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		photo := Photo{}
		err := json.NewDecoder(r.Body).Decode(&photo)
		if err != nil {
			log.Println("Could not decode photo", err)
			responses.BadRequest(w, r, "Could not decode photo")
			return
		}
		status, err := s.EditPhoto(&photo)
		if err != nil {
			responses.SwitchCase(w, r, status, err.Error())
			return
		}
		log.Println("photo", photo.ID, "updated successfully")
		responses.Success(w, r, "photo updated successfully")
	}
}

// DeletePhoto Delete a photo
func DeletePhoto(s PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			responses.BadRequest(w, r, "no ID in the query")
			return
		}
		confirm := r.URL.Query().Get("confirm")
		if confirm == "" {
			responses.BadRequest(w, r, "no confirm hash in the query")
			return
		}
		status, err := s.SafeDeletePhoto(id, confirm)
		if err != nil {
			responses.SwitchCase(w, r, status, err.Error())
			return
		}
		log.Println("Photo", id, "deleted successfully")
		responses.NoContent(w)
	}
}

// GetPhotos Get a list of photos
func GetPhotos(s PhotoService) http.HandlerFunc {
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
		// if r.Header.Get("Content-Type") == "" {
		// 	// responses.SendComponent(w, r, PhotoCards(photos))
		// } else {
		// 	responses.StructOK(w, r, photos)
		// }
	}
}
