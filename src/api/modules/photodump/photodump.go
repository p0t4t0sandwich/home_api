package photodump

import (
	"github.com/kolesa-team/goexiv"
	"home_api/src/database"
	"home_api/src/responses"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
)

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
