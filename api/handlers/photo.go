package handlers

import (
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
)

var mimes = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/gif":  "gif",
}

func GetPhoto(c *gin.Context) {
	// tmp dir
	dir := os.Getenv("TMP_PATH")
	err := os.MkdirAll(dir+"/original", os.ModePerm)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	os.MkdirAll(dir+"/100x100", os.ModePerm)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	//defer os.RemoveAll(dir)

	// Multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Form error: %s", err.Error()))
		return
	}

	files := form.File["file"]
	for _, file := range files {
		u, _ := uuid.NewV4()
		// save original file
		filepathOriginal := dir + "/original/" + u.String() + path.Ext(file.Filename)
		if err := c.SaveUploadedFile(file, filepathOriginal); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Upload file err: %s", err.Error()))
			return
		}

		// make thumb
		filepathThumb := dir + "/100x100/" + u.String() + path.Ext(file.Filename)
		err = ResizeExternally(filepathOriginal, filepathThumb, 100, 100)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}

	// get image from url
	urls := form.Value["url"]

	for _, url := range urls {

		response, err := http.Get(url)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
		defer response.Body.Close()

		u, _ := uuid.NewV4()

		//open a file for writing
		file, err := os.Create(dir + "/original/" + u.String() + path.Ext(url))
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
		defer file.Close()

		// Use io.Copy to just dump the response body to the file. This supports huge files
		_, err = io.Copy(file, response.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		// make thumb
		filepathThumb := dir + "/100x100/" + u.String() + path.Ext(url)
		err = ResizeExternally(file.Name(), filepathThumb, 100, 100)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}

	// get image from base64

	b64images := form.Value["base64image"]
	for _, v := range b64images {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		ext := mimes[http.DetectContentType(decoded)]

		u, _ := uuid.NewV4()

		//open a file for writing
		file, err := os.Create(dir + "/original/" + u.String() + "." + ext)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
		defer file.Close()

		_, err = file.Write(decoded)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		// make thumb
		filepathThumb := dir + "/100x100/" + u.String() + "." + ext
		err = ResizeExternally(file.Name(), filepathThumb, 100, 100)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

	}

	c.JSON(http.StatusOK, true)
}

func ResizeExternally(from string, to string, width uint, height uint) error {
	var args = []string{
		"--size", strconv.FormatUint(uint64(width), 10) + "x" +
			strconv.FormatUint(uint64(height), 10),
		"--output", to,
		"--crop",
		from,
	}
	lookPath, err := exec.LookPath("vipsthumbnail")
	if err != nil {
		return err
	}
	cmd := exec.Command(lookPath, args...)
	return cmd.Run()
}
