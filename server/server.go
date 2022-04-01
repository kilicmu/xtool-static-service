package server

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

const MAX_ONCE_UPLOAD_SIZE = 32 << 20
const STATIC_SOURCE_PATH = "static"
const SERVER_PORT = ":9090"

func safeDirPath() (string, error) {
	current := time.Now()
	year := strconv.Itoa(current.Year())
	month := current.Month().String()
	day := strconv.Itoa(current.Day())
	dPath := path.Join("./", STATIC_SOURCE_PATH, year, month, day)
	err := os.MkdirAll(dPath, 0750)
	return dPath, err
}

type UploadRespData struct {
	Md5  string `json:"md5"`
	Path string `json:"path"`
	Ext  string `json:"ext"`
}

func uploadHandler(context *gin.Context) {
	r := context.Request
	w := context.Writer
	fmt.Println("upload form", r.Host, r.Method)
	err := r.ParseMultipartForm(MAX_ONCE_UPLOAD_SIZE)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("parse multipartForm error ", err)
		return
	}

	files := r.MultipartForm.File
	headers := make([]*multipart.FileHeader, 0)
	for _, hs := range files {
		headers = append(headers, hs...)
	}
	ch := make(chan UploadRespData, len(headers))
	dirPath, err := safeDirPath()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, h := range headers {
		go func(curH *multipart.FileHeader) {
			rawFilename := curH.Filename
			f, _ := curH.Open()

			defer f.Close()
			chunk := make([]byte, 0)

			for {
				buf := make([]byte, 1024)
				n, err := f.Read(buf)
				if err == io.EOF {
					break
				}
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					log.Fatal("read file error", err)
					return
				}
				chunk = append(chunk, buf[:n]...)
			}

			sum := md5.Sum(chunk)
			uploadFileMd5 := hex.EncodeToString(sum[:])
			ext := path.Ext(rawFilename)
			filename := uploadFileMd5 + ext
			cacher := context.MustGet("cacher").(*cache.Cache)

			guessCachedAssetPath, has := cacher.Get(filename)
			fmt.Println(has, guessCachedAssetPath, filename)
			if has {
				assetPath := guessCachedAssetPath.(string)
				ch <- UploadRespData{uploadFileMd5, "http://" + r.Host + "/" + assetPath, ext}
				return
			}

			assetPath := path.Join(dirPath, filename)
			cacher.Set(filename, assetPath, time.Hour*2)
			targetFs, err := os.OpenFile(assetPath, os.O_WRONLY|os.O_CREATE, 0777)
			defer targetFs.Close()
			_, err = targetFs.Write(chunk)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("write to target error ", err)
				return
			}
			ch <- UploadRespData{uploadFileMd5, "http://" + r.Host + "/" + assetPath, ext}
		}(h)
	}

	ret := make([]UploadRespData, 0)
	for len(ret) != len(headers) {
		select {
		case item := <-ch:
			ret = append(ret, item)
		}
	}
	b, err := json.Marshal(ret)
	if err != nil {
		log.Println("json parse error ", err)
	}
	w.Write(b)
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method)
	if r.Method != "GET" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	fmt.Println(r.RequestURI)
}

func Start() {
	router := gin.Default()
	router.Use(UseCacher())
	v0 := router.Group("/v0")
	{
		v0.POST("/upload", uploadHandler)
	}
	router.Static("/static", "./static")
	router.Run(":9090")
}
