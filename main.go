package main

import (
	"encoding/base64"
	"io/ioutil"
	"fmt"
	"log"
	"math"
	"net/http"
	"path"
	"os"
	"strings"
	"time"
	"unsafe"
)

type VersionInfo struct {
	Major int8
	Minor int8
	Patch int8
	Release string
}

const (
	Domain string = "https://someurl.com"
	FileRoute string = "/"
	FormName string = "file"
	MaxUploadSize int64 = 100
	Port string = ":56749"
	Secret string = "youshallnotpass"
	StoragePath string = "./uploads"
	UploadRoute string = "/upload"
)

var (
	Version VersionInfo = VersionInfo{
		Major: 1,
		Minor: 0,
		Patch: 2,
		Release: "stable",
	}
	Whitelist []string = []string{}
)

func getIP(r *http.Request) string {
	address := r.Header.Get("CF-Connecting-IP")
	if address == "" {
		address = r.Header.Get("True-Client-IP")
	}
	
	if address == "" {
		address = r.Header.Get("X-Real-IP")
	}

    if address == "" {
        address = r.Header.Get("X-Forwarded-For")
	}
	
    if address == "" {
        address = r.RemoteAddr
	}
	
    return address
}

func genName(str string) string {
	timeNano := time.Now().UnixNano()
	return base64.RawURLEncoding.EncodeToString((*[8]byte)(unsafe.Pointer(&timeNano))[:]) + path.Ext(str)
}

func isWhitelistedIP(r *http.Request) bool {
	for _, ip := range Whitelist {
		if ip == getIP(r) {
			return true
		}
	}

	return false
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if isWhitelistedIP(r) || len(Whitelist) == 0 { 
		providedSecret := r.Header.Get("Authorization")
		if providedSecret != Secret {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			log.Printf("%s: Attempted to upload file with incorrect secret: %s", getIP(r), providedSecret)
			return
		}
		
		r.ParseMultipartForm(MaxUploadSize)
		file, handler, err := r.FormFile(FormName)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Fatalf("Error when reading form file: %s", err.Error())
			return
		}		

		if float64(handler.Size) > (float64(MaxUploadSize) * math.Pow(1024, 2)) {
			http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
			log.Printf("%s: Attempted to upload file with size: %d", getIP(r), handler.Size)
			return
		}
		defer file.Close()

		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Fatalf("Error when reading from uploaded file: %s", err.Error())
			return
		}

		name := genName(handler.Filename)
		_ = os.Mkdir(StoragePath, 0700)
		err = ioutil.WriteFile(StoragePath + "/" + name, fileBytes, 0644)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Fatalf("Error when writing to new file: %s", err.Error())
			return
		}

		url := Domain
		if !strings.HasSuffix(Domain, "/") {
			url += "/"
		} 
		if FileRoute != "/" {
			url += FileRoute
		}
		url += name

		fmt.Fprint(w, url)
		log.Printf("%s: Uploaded file %s", getIP(r), name)
		return
	}

	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	log.Printf("%s: Attempted to upload file", getIP(r))
	return
}

func main() {
	log.Printf("Running goshare v%d.%d.%d-%s", Version.Major, Version.Minor, Version.Patch, Version.Release)
	
	fs := http.FileServer(http.Dir(StoragePath))
	if FileRoute == "/" {
		http.Handle(FileRoute, fs)
	} else {
		http.Handle(FileRoute, http.StripPrefix(FileRoute, fs))
	}

	http.HandleFunc(UploadRoute, uploadFile)
	http.ListenAndServe(Port, nil)
}