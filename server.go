package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type DiffImage struct {
	Number   int    // page number
	Filename string // file1
}

type ResultPage struct {
	Hash1       string
	Hash2       string
	Differences []DiffImage
}

func indexController(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		fmt.Println("A new request has been made on / but the method " + r.Method + " was not supported.")
		return
	}

	// TODO (idea): list pdf on upload folders

	// Display the compare page
	t, _ := template.ParseFiles("templates/index.html")
	t.Execute(w, nil)

}

func compareController(w http.ResponseWriter, r *http.Request) {
	// Set a limit of 32 MB per request
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)

	if r.Method == "GET" {
		// Redirect to index page
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	} else if r.Method == "POST" {
		parseErr := r.ParseMultipartForm(32 << 20)
		if parseErr != nil {
			http.Error(w, "failed to parse multipart message", http.StatusBadRequest)
			return
		}

		if len(r.MultipartForm.File) != 2 {
			http.Error(w, "two file pdfs per comparision", http.StatusBadRequest)
			return
		}

		// Grab the two PDF(s) from the form
		pdfFile1 := r.MultipartForm.File["pdf-1"]
		pdfFile2 := r.MultipartForm.File["pdf-2"]

		// Check if the two files are PDF

		file1, err := pdfFile1[0].Open()
		if err != nil {
			panic(err)
		}
		defer file1.Close()

		buff := make([]byte, 512)
		if _, err = file1.Read(buff); err != nil {
			panic(err)
		}

		var pdf1hash string

		if http.DetectContentType(buff) == "application/pdf" {
			out, err := os.Create("data/uploads/" + filepath.Clean(pdfFile1[0].Filename))
			if err != nil {
				panic(err)
			}
			_, err = file1.Seek(0, io.SeekStart)
			if err != nil {
				panic(err)
			}
			io.Copy(out, file1)
			pdf1hash = ComputeSha256("data/uploads/" + filepath.Clean(pdfFile1[0].Filename))
		}

		file2, err := pdfFile2[0].Open()
		if err != nil {
			panic(err)
		}
		defer file2.Close()
		if _, err = file2.Read(buff); err != nil {
			panic(err)
		}

		var pdf2hash string

		if http.DetectContentType(buff) == "application/pdf" {
			// Write them in upload folder
			out, err := os.Create("data/uploads/" + filepath.Clean(pdfFile2[0].Filename))
			if err != nil {
				panic(err)
			}
			_, err = file2.Seek(0, io.SeekStart)
			if err != nil {
				panic(err)
			}
			io.Copy(out, file2)
			pdf2hash = ComputeSha256("data/uploads/" + filepath.Clean(pdfFile2[0].Filename))
		}

		fmt.Println("Starting a new job....")

		// Start the job

		hexToRGB("ff2010")
		go CreatePNG("data/uploads/" + filepath.Clean(pdfFile1[0].Filename))
		go CreatePNG("data/uploads/" + filepath.Clean(pdfFile2[0].Filename))
		go Compare("data/uploads/"+filepath.Clean(pdfFile1[0].Filename), "data/uploads/"+filepath.Clean(pdfFile2[0].Filename))

		// Redirect to result page
		http.Redirect(w, r, "/compare/"+pdf1hash+"-"+pdf2hash, http.StatusMovedPermanently)

	} else {
		fmt.Println("A new request has been made on /compare but the method " + r.Method + " was not supported.")
		return
	}

}

func retrieveFilesController(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Path[len("/compare/"):]
	if len(slug) == 0 {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}
	hashes := strings.Split(slug, "-")
	fmt.Printf("%s ", hashes[0])
	fmt.Printf("%s \n", hashes[1])

	// Checks if the folder were already created

	if _, err := os.Stat("data/generated/" + hashes[0]); errors.Is(err, os.ErrNotExist) {
		http.Error(w, "The two pdfs ("+hashes[0]+", "+hashes[1]+") were not compared.", http.StatusNotFound)
		return
	}

	if _, err := os.Stat("data/" + hashes[0] + "/.tmp"); errors.Is(err, os.ErrExist) {
		http.Error(w, "The images are being created. It should take a few seconds.", http.StatusOK)
		return
	}

	if _, err := os.Stat("data/generated/" + hashes[0] + "/.tmp"); errors.Is(err, os.ErrExist) {
		http.Error(w, "pdf-diff takes a while to generate all the images.", http.StatusOK)
		return
	}

	// Checks the result generated

	// List all the images given a filename (e.g. filename-1.png)

	f, err := os.Open("data/generated/" + hashes[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	files, err := f.Readdir(0)
	if err != nil {
		fmt.Println(err)
		return
	}
	i := 0
	var differences []DiffImage

	for _, v := range files {
		single := DiffImage{
			Number:   i,
			Filename: v.Name(),
		}
		differences = append(differences, single)
		i++
	}

	structure := ResultPage{
		Hash1:       hashes[0],
		Hash2:       hashes[1],
		Differences: differences,
	}

	t := template.Must(template.ParseFiles("templates/result.html"))
	if err != nil {
		panic(err)
	}

	err = t.Execute(w, structure)
	if err != nil {
		panic(err)
	}
}

func StartServer() {

	http.HandleFunc("/", indexController)
	http.HandleFunc("/compare", compareController)
	http.HandleFunc("/compare/", retrieveFilesController)

	http.Handle("/results/", http.StripPrefix("/results/", http.FileServer(http.Dir("./data"))))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}

}
