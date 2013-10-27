package main

import (
    "fmt"
    "log"
    "labix.org/v2/mgo"
    "net/http"
) 
	
var peopleContainer *mgo.Collection
var records []Record

type Record struct {
    FirstName string
    MiddleName string
    LastName string
    Text string
}

func familyTreeHandler(w http.ResponseWriter, r *http.Request) {
    iter := peopleContainer.Find(nil).Sort("lastname").Iter()

    err := iter.All(&records)

    if err != nil {
        log.Fatal(err)
    }

    fmt.Fprintf(w, "<html><head></head><body>")

    for _, r := range(records) {
        fmt.Fprintf(w, "<B>")
        fmt.Fprintf(w, "Name: %s %s %s\n", r.FirstName, r.MiddleName, r.LastName)
        fmt.Fprintf(w, "</B>")
        fmt.Fprintf(w, "<br><B>Description:</B> %s\n", r.Text)
        fmt.Fprintf(w, "<hr>")
    }
    fmt.Fprintf(w, "</body></html>")

}

func main() {

    session, err := mgo.Dial("localhost")

    if err != nil {
        log.Fatal(err)
    }

    defer session.Close()

    session.SetMode(mgo.Monotonic, true)

    peopleContainer = session.DB("genealogy").C("people")

    http.HandleFunc("/dulaney", familyTreeHandler) 

    log.Fatal(http.ListenAndServe(":80", nil))
}
