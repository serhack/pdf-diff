package main

import (
	"fmt"
	"net/http"
	"html/template"
)

func indexController(w http.ResponseWriter, r *http.Request){
	if r.Method != "GET"{
		fmt.Println("A new request has been made on / but the method " + r.Method + " was not supported.")
		return
	}

	// Display the compare page
	t, _ := template.ParseFiles("templates/index.html")
	t.Execute(w, nil)

}

func compareController(w http.ResponseWriter, r *http.Request){

	if r.Method == "GET"{
		// Redirect to index page
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	} else if r.Method == "POST"{
		// Grab the two PDF(s) from the form
		

		// Check if it's a real PDF

		// Sanitize the name of the pdf

		// Write them in upload folder

		// Start the job

		// Redirect to result page
	} else {
		fmt.Println("A new request has been made on /compare but the method " + r.Method + " was not supported.")
		return
	}

}

func StartServer(){

	http.HandleFunc("/", indexController)
	http.HandleFunc("/compare", compareController)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}

}