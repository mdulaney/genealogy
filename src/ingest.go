package main

import (
    "code.google.com/p/go.net/html"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "labix.org/v2/mgo"
    "path"
    "strings"
)

type Record struct {
    FirstName string
    MiddleName string
    LastName string
    Text string
}

func ProcessRecords(n *html.Node) []Record {

    records := make([]Record, 0)

    r := Record { }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        if c.Type == html.ElementNode && c.Data == "b" {
            name := strings.TrimRight(strings.TrimLeft(c.FirstChild.Data, " "), " ")
            name = strings.Replace(name, "\n", " ", -1)

            names := strings.Split(name, " ")

            if len(names) == 2 {
                r.FirstName = names[0]
                r.LastName = names[1]
            } else if len(names) == 3 {
                r.FirstName = names[0]
                r.MiddleName = names[1]
                r.LastName = names[2]
            } else {
                r.FirstName = names[0]
            }

            fmt.Printf("Name: %s\n", name)
            r.Text += name
        } else if c.Type != html.ElementNode {
            r.Text += c.Data
        }

        if c.Type == html.ElementNode && c.Data == "hr" {
            r.Text = strings.Replace(r.Text, "\n", " ", -1)
            fmt.Printf("Description: %s\n", r.Text)
            fmt.Printf("-------------\n")
            records = append(records, r)
            r = Record { }
        } else if c.Type == html.ElementNode && c.Data == "p" {
            for c2 := c.FirstChild; c2 != nil; c2 = c2.NextSibling {
                if c2.Type == html.ElementNode && c2.Data == "a" {
                    if c2.FirstChild.Type != html.ElementNode {
                        r.Text += c2.FirstChild.Data
                    }
                } else if c2.Type != html.ElementNode {
                    r.Text += c2.Data
                }
            }
        } else if c.Type == html.ElementNode && c.Data == "a" {
            for c2 := c.FirstChild; c2 != nil; c2 = c2.NextSibling {
                if c2.Type != html.ElementNode {
                    r.Text += c2.Data
                }
            }

        }
    }

    // TODO: this is redundant
    r.Text = strings.Replace(r.Text, "\n", " ", -1)
    fmt.Printf("Description: %s\n", r.Text)
    fmt.Printf("-------------\n")

    records = append(records, r)
    return records
}

func ProcessDoc(n *html.Node) []Record {

    for c := n.FirstChild; c != nil; c = c.NextSibling {
        if c.Type == html.ElementNode && c.Data == "html" {
            return ProcessDoc(c)
        } else if c.Type == html.ElementNode && c.Data == "body" {
            return ProcessRecords(c)
        }
    }

    return nil
}

func main() {

    dirName := flag.String("d", "", "directory name")
    mongoHost := flag.String("t", "", "mongo host")

    flag.Parse()

    records := make([]Record, 0)

    if *dirName == "" {
        log.Fatal("Error: must specify directory name\n")
    }

    if *mongoHost == "" {
        log.Fatal("Error: must specify mongo host\n")
    }

    files, err := ioutil.ReadDir(*dirName)

    if err != nil {
        log.Fatal(err)
    }

    for _, fi := range(files) {

        if !strings.HasSuffix(fi.Name(), ".htm") {
            continue
        }

        htmlText, err := ioutil.ReadFile(path.Join(*dirName, fi.Name()))

        if err != nil {
            log.Fatal(err)
        }

        doc, err := html.Parse(strings.NewReader(string(htmlText)))
        if err != nil {
            log.Fatal(err)
        }

        records = append(records, ProcessDoc(doc)...)
    }

    fmt.Printf("Processed %d records\n", len(records))

    fmt.Printf("Connecting to mongo host: %s\n", *mongoHost)
    session, err := mgo.Dial(*mongoHost)

    if err != nil {
        log.Fatal(err)
    }

    defer session.Close()

    session.SetMode(mgo.Monotonic, true)

    fmt.Printf("Starting session\n")
    c := session.DB("genealogy").C("people")

    for _, r := range(records) {
        err = c.Insert(&r)
    }

}
