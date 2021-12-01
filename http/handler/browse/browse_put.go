package browse

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/portainer/agent/filesystem"
	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/libhttp/response"
)

type browsePutPayload struct {
	Path       string
	Filename   string
	File       []byte
	Fileheader *multipart.FileHeader
}

func (payload *browsePutPayload) Validate(r *http.Request) error {
	path, err := request.RetrieveMultiPartFormValue(r, "Path", false)
	if err != nil {
		return errors.New("Invalid file path")
	}
	payload.Path = path

	file, headers, err := r.FormFile("file")
	if err != nil {
		return errors.New("Invalid uploaded file")
	}
	defer file.Close()
	payload.Fileheader = headers
	payload.Filename = headers.Filename

	return nil
}

// POST request on /browse/put?volumeID=:id
func (handler *Handler) browsePut(rw http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	fmt.Println("here comes a request")
	var payload browsePutPayload
	values := r.URL.Query()
	volumeID := values.Get("volumeID")
	r.ParseMultipartForm(1024 * 1024 * 32)
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		if fhs := r.MultipartForm.File["file"]; len(fhs) > 0 {
			f, err := fhs[0].Open()
			if err != nil {
				return &httperror.HandlerError{http.StatusBadRequest, "Invalid file", err}
			}
			defer f.Close()
			//return f, fhs[0], err
			payload.Fileheader = fhs[0]
			payload.Filename = payload.Fileheader.Filename

			fmt.Printf("payload filename is %s !\n", payload.Filename)
		}
	} else {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid request payload", errors.New("Invalid uploaded file")}
	}
	payload.Path = "/"
	if vs := r.Form["Path"]; len(vs) > 0 {
		payload.Path = vs[0]
		fmt.Printf("file path is %s !\n", payload.Path)
	} else {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid request payload", errors.New("Invalid file path")}
	}

	fmt.Println("ready to transfer")

	err := errors.New("")
	if volumeID != "" {
		payload.Path, err = filesystem.BuildPathToFileInsideVolume(volumeID, payload.Path)
		if err != nil {
			return &httperror.HandlerError{http.StatusBadRequest, "Invalid volume", err}
		}

		_, err = filesystem.BuildPathToFileInsideVolume(volumeID, payload.Filename)
		if err != nil {
			return &httperror.HandlerError{http.StatusBadRequest, "Invalid filename", err}
		}
	}

	err = filesystem.WriteBigFile(payload.Path, payload.Filename, payload.Fileheader, 0755)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Error saving file to disk", err}
	}
	return response.Empty(rw)
}

// POST request on /v1/browse/:id/put
func (handler *Handler) browsePutV1(rw http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	volumeID, err := request.RetrieveRouteVariableValue(r, "id")
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid volume identifier route variable", err}
	}

	var payload browsePutPayload
	err = payload.Validate(r)
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid request payload", err}
	}
	payload.Path, err = filesystem.BuildPathToFileInsideVolume(volumeID, payload.Path)
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid volume", err}
	}

	err = filesystem.WriteFile(payload.Path, payload.Filename, payload.File, 0755)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Error saving file to disk", err}
	}

	return response.Empty(rw)
}
