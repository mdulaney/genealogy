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

type Child struct {
    Identifier string
    Name string
}

type Marriage struct {
    OtherIdentifier string
    OtherName string
    Children []*Child
}

type Parent struct {
    Identifier string
    Name string
}

type Record struct {
    FirstName string
    MiddleName string
    LastName string
    Identifier string
    Text string
    Marriages []*Marriage
    Parents [2]*Parent
    curMarriageIdx int
}

type DocumentContext struct {
    s DocumentState
    cur *html.Node
    curRecord *Record
    records []*Record
}

func NewRecord() *Record {
    r := new(Record)
    r.Marriages = make([]*Marriage, 0)
    return r
}

func (dc *DocumentContext) ChangeState(s DocumentState) {
    dc.s = s
}

type DocumentState interface {
    HandleATag(dc *DocumentContext)
    HandleName(dc *DocumentContext)
    HandleDescription(dc *DocumentContext)
    HandleDone(dc *DocumentContext)
    String() string
}

func NewDocumentContext(n *html.Node) *DocumentContext {
    ctx := new(DocumentContext)

    ctx.s = &HandlingIdentifier{}
    ctx.cur = n.FirstChild
    ctx.records = make([]*Record, 0)

    return ctx
}

func PrintTag(level int, ds DocumentState, n *html.Node) {
    fmt.Printf("%-12s", ds)
    for i := 1; i < level; i++ {
        fmt.Printf("---|")
    }

    fmt.Printf("--->")

    if  n.Type == html.ElementNode || n.Type == html.DocumentNode || n.Type == html.DoctypeNode {
        fmt.Printf(" <%s>\n", n.Data)
    } else {
        fmt.Printf(" <%s>\n", "Body")
    }
}

func PrintData(level int, ds DocumentState, s string) {
    fmt.Printf("%-12s", ds)

    for i := 1; i < level; i++ {
        fmt.Printf("---|")
    }

    fmt.Printf("--->")

    fmt.Printf(" (%s)\n", s)
}

func (dc *DocumentContext) ProcessDocument() {

    for ; dc.cur != nil ; {

        PrintTag(1, dc.s, dc.cur)
        if dc.cur.Type == html.DoctypeNode && dc.cur.Data == "html" {
            dc.cur = dc.cur.NextSibling
        } else if dc.cur.Type == html.ElementNode {
            if dc.cur.Data == "html" {
                dc.cur = dc.cur.FirstChild
            } else if dc.cur.Data == "body" {
                dc.cur = dc.cur.FirstChild
            } else if dc.cur.Data == "head" {
                dc.cur = dc.cur.NextSibling
            } else if dc.cur.Data == "a" {

                // Catches the end of the document
                for _, a := range dc.cur.Attr {
                    if a.Key == "href" && strings.HasSuffix(a.Val, ".htm") {
                        dc.s.HandleDone(dc)
                        return
                    }
                }
                dc.s.HandleATag(dc)
            } else if dc.cur.Data == "b" {
                dc.s.HandleName(dc)
            } else if dc.cur.Data == "sup" {
                dc.s.HandleDescription(dc)
            } else if dc.cur.Data == "hr" {
                dc.s.HandleDone(dc)
            } else {
                // Encountered a tag we don't care about.  Increment the state
                dc.cur = dc.cur.NextSibling
            }
        } else {
            // increments the state
            dc.s.HandleDescription(dc)
        }
    }
}

type HandlingIdentifier struct { }
type HandlingDescription struct { }
type HandlingParents struct { }
type HandlingMarried struct { }
type HandlingChildren struct { }

func ProcessPersonIdentifier(n *html.Node) string {
    for _, a := range(n.Attr) {
        if a.Key != "name" {
            log.Fatal("Error: expected `name` tag attribute")
        } else {
            return a.Val
        }
    }
    return ""
}

func ProcessAncestorReference(n *html.Node) string {
    for _, a := range(n.Attr) {
        if a.Key != "href" {
            log.Fatal("Error: expected `href` tag attribute")
        } else {
            return strings.Split(a.Val, "#")[1]
        }
    }
    return ""
}

// Handlingidentifier
func (hi *HandlingIdentifier) HandleATag(ds *DocumentContext) {
    ds.curRecord = NewRecord()

    // process identifier
    ds.curRecord.Identifier = ProcessPersonIdentifier(ds.cur)

    if ds.curRecord.Identifier == "" {
        log.Fatal("Error: identifier not found\n")
    }

    PrintData(2, ds.s, ds.curRecord.Identifier)
    ds.cur = ds.cur.NextSibling
}

func (hi *HandlingIdentifier) String() string {
    return "Identifier"
}

func (hi *HandlingIdentifier) HandleName(dc *DocumentContext) {

    PrintTag(2, dc.s, dc.cur.FirstChild)

    name := strings.TrimRight(strings.TrimLeft(dc.cur.FirstChild.Data, " "), " ")
    name = strings.Replace(name, "\n", " ", -1)

    PrintData(3, dc.s, name)

    names := strings.Split(name, " ")

    if len(names) == 2 {
        dc.curRecord.FirstName = names[0]
        dc.curRecord.LastName = names[1]
    } else if len(names) == 3 {
        dc.curRecord.FirstName = names[0]
        dc.curRecord.MiddleName = names[1]
        dc.curRecord.LastName = names[2]
    } else {
        dc.curRecord.FirstName = names[0]
    }
    dc.cur = dc.cur.NextSibling
    dc.ChangeState(&HandlingDescription{})
}

func (hi *HandlingIdentifier) HandleDescription(ds *DocumentContext) {
    if strings.TrimSpace(ds.cur.Data) != "" {
        log.Fatal("Error: unexpected tag")
    }

    ds.cur = ds.cur.NextSibling
}

func (hi *HandlingIdentifier) HandleDone(ds *DocumentContext) {
    log.Fatal("Error: unexpected tag")
}


// HandlingDescription
func (hd *HandlingDescription) HandleATag(ds *DocumentContext) {
    log.Fatal("Error: unexpected tag")
}

func (hd *HandlingDescription) HandleName(ds *DocumentContext) {
    log.Fatal("Error: unexpected tag")
}

func (hd *HandlingDescription) HandleDescription(dc *DocumentContext) {
    curBodyString := dc.cur.Data
    PrintData(2, dc.s, curBodyString)
    if strings.Contains(curBodyString, "Parents:") {
        dc.ChangeState(&HandlingParents{})
    } else if strings.Contains(curBodyString, "was married to") {
        dc.ChangeState(&HandlingMarried{})
    } else if strings.Contains(curBodyString, "Children were:") {
        dc.ChangeState(&HandlingChildren{})
    } else {
        dc.curRecord.Text += curBodyString
    }

    dc.cur = dc.cur.NextSibling
}

func (hd *HandlingDescription) HandleDone(ds *DocumentContext) {
    ds.records = append(ds.records, ds.curRecord)
    ds.cur = ds.cur.NextSibling
    ds.ChangeState(&HandlingIdentifier{})
}

func (hi *HandlingDescription) String() string {
    return "Description"
}

// HandlingParents
func (hm *HandlingParents) HandleATag(ds *DocumentContext) {

    var pIdentifier string

    pIdentifier = ProcessAncestorReference(ds.cur)

    // Process Parents
    PrintData(2, ds.s, pIdentifier)
    PrintTag(2, ds.s, ds.cur.FirstChild)          // <body>
    PrintData(3, ds.s, ds.cur.FirstChild.Data)    // name

    ds.curRecord.Parents[0] = new(Parent)
    ds.curRecord.Parents[0].Name = ds.cur.FirstChild.Data
    ds.curRecord.Parents[0].Identifier = pIdentifier

    ds.cur = ds.cur.NextSibling

    if strings.TrimSpace(ds.cur.Data) == "and" {
        PrintTag(1, ds.s, ds.cur)                     // <body>
        PrintData(2, ds.s, ds.cur.Data)               // 'and'

        ds.cur = ds.cur.NextSibling

        PrintTag(1, ds.s, ds.cur)                     // <a>

        pIdentifier = ProcessAncestorReference(ds.cur)
        PrintData(2, ds.s, pIdentifier)               // identifier

        PrintTag(2, ds.s, ds.cur.FirstChild)          // <body>
        PrintData(3, ds.s, ds.cur.FirstChild.Data)    // name

        ds.curRecord.Parents[1] = new(Parent)
        ds.curRecord.Parents[1].Name = ds.cur.FirstChild.Data
        ds.curRecord.Parents[1].Identifier = pIdentifier

        ds.cur = ds.cur.NextSibling                   // next
    }

    ds.ChangeState(&HandlingDescription{})
}

func (hm *HandlingParents) HandleName(ds *DocumentContext) {
    log.Fatal("Error: unexpected tag")
}

func (hm *HandlingParents) HandleDescription(ds *DocumentContext) {
    log.Fatal("Error: unexpected tag")
}

func (hm *HandlingParents) HandleDone(ds *DocumentContext) {
    ds.cur = ds.cur.NextSibling
    ds.ChangeState(&HandlingIdentifier{})
}

func (hi *HandlingParents) String() string {
    return "Parents"
}

// HandlingMarried
func (hm *HandlingMarried) HandleATag(ds *DocumentContext) {
    m := new(Marriage)

    othIdentifier := ProcessAncestorReference(ds.cur)

    PrintData(2, ds.s, othIdentifier)
    PrintTag(2, ds.s, ds.cur.FirstChild)
    PrintData(3, ds.s, ds.cur.FirstChild.Data)

    m.OtherName = ds.cur.FirstChild.Data
    m.OtherIdentifier = othIdentifier

    ds.curRecord.Marriages = append(ds.curRecord.Marriages, m)
    ds.curRecord.curMarriageIdx++

    ds.cur = ds.cur.NextSibling

    ds.ChangeState(&HandlingDescription{})
}

func (hm *HandlingMarried) HandleName(ds *DocumentContext) {
    log.Fatal("Error: unexpected tag")
}

func (hm *HandlingMarried) HandleDescription(ds *DocumentContext) {
    log.Fatal("Error: unexpected tag")
}

func (hm *HandlingMarried) HandleDone(ds *DocumentContext) {
    ds.ChangeState(&HandlingIdentifier{})
}

func (hi *HandlingMarried) String() string {
    return "Married"
}

// HandlingChildren
func (hi *HandlingChildren) HandleATag(ds *DocumentContext) {

    var marriage *Marriage
    var cIdentifier string

    rec := ds.curRecord

    if rec.curMarriageIdx == 0 {
        marriage = new(Marriage)
        rec.Marriages = append(rec.Marriages, marriage)
    } else {
        marriage = rec.Marriages[rec.curMarriageIdx-1]
    }

    // display the reference id
    cIdentifier = ProcessAncestorReference(ds.cur)
    PrintData(2, ds.s, cIdentifier)

    // consume the a href
    PrintTag(2, ds.s, ds.cur.FirstChild)
    PrintData(3, ds.s, ds.cur.FirstChild.Data)

    // create new child
    c := new(Child)
    c.Name = ds.cur.FirstChild.Data
    c.Identifier = cIdentifier

    marriage.Children = append(marriage.Children, c)

    ds.cur = ds.cur.NextSibling

    for ;; {
        if ds.cur.Type == html.ElementNode || strings.TrimSpace(ds.cur.Data) != "," {
            ds.ChangeState(&HandlingDescription{})
            return

        } else {
            // consume the comma
            ds.cur = ds.cur.NextSibling
        }

        // consume the a href
        PrintTag(1, ds.s, ds.cur)

        // consume the reference identifier
        cIdentifier = ProcessAncestorReference(ds.cur)
        PrintData(2, ds.s, cIdentifier)

        // consume the name
        PrintTag(2, ds.s, ds.cur.FirstChild)
        PrintData(3, ds.s, ds.cur.FirstChild.Data)

        // create new child
        c := new(Child)
        c.Name = ds.cur.FirstChild.Data
        c.Identifier = cIdentifier

        marriage.Children = append(marriage.Children, c)

        ds.cur = ds.cur.NextSibling
    }
}

func (hi *HandlingChildren) HandleName(ds *DocumentContext) {
    log.Fatal("Error: %s unexpected tag")
}

func (hi *HandlingChildren) HandleDescription(ds *DocumentContext) {
    log.Fatal("Error: unexpected tag")
}

func (hi *HandlingChildren) HandleDone(ds *DocumentContext) {
    ds.records = append(ds.records, ds.curRecord)
    ds.cur = ds.cur.NextSibling
    ds.ChangeState(&HandlingIdentifier{})
}

func (hi *HandlingChildren) String() string {
    return "Children"
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

func main() {

    dirName := flag.String("d", "", "directory name")
    mongoHost := flag.String("t", "", "mongo host")

    flag.Parse()

    records := make([]*Record, 0)

    if *dirName == "" {
        log.Fatal("Error: must specify directory name\n")
    }

    files, err := ioutil.ReadDir(*dirName)

    if err != nil {
        log.Fatal(err)
    }

    for _, fi := range(files) {

        if !strings.HasSuffix(fi.Name(), ".htm") {
            continue
        }

        fmt.Printf("------------ %s -------------\n", fi.Name())
        htmlText, err := ioutil.ReadFile(path.Join(*dirName, fi.Name()))

        if err != nil {
            log.Fatal(err)
        }

        // Remove the paragraph tags to simplify later processing steps
        normal := strings.Replace(string(htmlText), "<P>", "", -1)
        normal = strings.Replace(normal, "\n", " ", -1)

        doc, err := html.Parse(strings.NewReader(normal))
        if err != nil {
            log.Fatal(err)
        }

        ctx := NewDocumentContext(doc)
        ctx.ProcessDocument()

        records = append(records, ctx.records...)
    }

    fmt.Printf("Processed %d records\n", len(records))

    if *mongoHost != "" {
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

}
