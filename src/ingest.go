package main

import (
    "code.google.com/p/go.net/html"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    _ "labix.org/v2/mgo"
    "path"
    "strconv"
    "strings"
    "time"
)

const MAX_MONTH_DAYS = 31

var monthMap = map[string]time.Month {
    "Jan" : time.January,
    "Feb" : time.February,
    "Mar" : time.March,
    "Apr" : time.April,
    "May" : time.May,
    "Jun" : time.June,
    "Jul" : time.July,
    "Aug" : time.August,
    "Sep" : time.September,
    "Oct" : time.October,
    "Nov" : time.November,
    "Dec" : time.December,
}
type Location struct {
    county string
    state string
    town string
}

type Date struct {
    year int
    month time.Month
    day int
}

type DatedEvent struct {
    d Date
    loc Location
}

type Child struct {
    Identifier string
    Name string
}

type Marriage struct {
    OtherIdentifier string
    OtherName string
    Children []*Child
    Date *DatedEvent
}

type Parent struct {
    Identifier string
    Name string
}

type Gender int

const (
    Male = iota
    Female = iota
)

type Occupation struct {
    Name string
    Date *DatedEvent
}

type Description struct {
    Text string
    Date *DatedEvent
}

type Residence struct {
    Date *DatedEvent
}

type Record struct {
    FirstName string
    MiddleName string
    LastName string
    Identifier string
    Text string
    Marriages []*Marriage
    Parents [2]*Parent
    Children []*Child
    BirthDate *DatedEvent
    Census *DatedEvent
    Death *DatedEvent
    Residences []*Residence
    residenceIdx int
    Alias string
    Occ *Occupation
    Desc *Description
    gender Gender
    curMarriageIdx int
}

func NewRecord() *Record {
    rec := new(Record)
    rec.Children = make([]*Child, 0)
    return rec
}
type Document struct {
    Paragraphs []*Paragraph
}

type Paragraph struct {
    Data string
    Frags []*Frag
    NormalizedFrags []*Frag
    Sentences []*Sentence
}

type Sentence struct {
    Frags []*Frag
}

func (s *Sentence) String() string {
    var allStr string
    allWords := s.AllWords()

    for _, a := range(allWords) {
        allStr += a
        allStr += " "
    }

    return strings.TrimSpace(allStr)
}

func (s *Sentence) AllWords() []string {
    allWords := make([]string, 0)

    for _, f := range(s.Frags) {
        for _, w := range(strings.Split(f.Data, " ")) {
            allWords = append(allWords, w)
        }
    }

    return allWords
}

func (s *Sentence) Contains(str string) bool {

    var one string
    all := s.AllWords()

    for _, a := range(all) {
        one += a
        one += " "
    }

    one = strings.TrimSpace(one)

    return strings.Contains(one, str)
}

type Frag struct {
    Data string
    RefId string
    Identifier string
    IsSup bool
}

func PrintTag(level int, n *html.Node) {
    //fmt.Printf("%-12s", ds)
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

func ProcessPersonIdentifier(n *html.Node) string {
    for _, a := range(n.Attr) {
        if a.Key != "name" {
            log.Fatal("Error: expected `name` tag attribute")
        } else {
            return strings.TrimSpace(a.Val)
        }
    }
    return ""
}

func ProcessAncestorReference(n *html.Node) (string, string) {
    for _, a := range(n.Attr) {
        if a.Key != "href" {
            log.Fatal("Error: expected `href` tag attribute")
        } else if !strings.HasSuffix(a.Val, ".htm") {
            return strings.TrimSpace(strings.Split(a.Val, "#")[1]),
                    n.FirstChild.Data
        }
    }
    return "", ""
}

func ProcessDocument(n *html.Node) *Document {
    var para *Paragraph = nil

    doc := new(Document)

    doc.Paragraphs = make([]*Paragraph, 0)

    curNode := n.FirstChild
    for ; curNode != nil ; {

        //PrintTag(1, curNode)
        if curNode.Type == html.DoctypeNode && curNode.Data == "html" {
            curNode = curNode.NextSibling
        } else if curNode.Type == html.ElementNode {
            if curNode.Data == "html" || curNode.Data == "body" {
                curNode = curNode.FirstChild
            } else if curNode.Data == "head" {
                curNode = curNode.NextSibling
            } else if curNode.Data == "a" {

                // Catches the end of the document
                for _, a := range curNode.Attr {
                    if a.Key == "href" && strings.HasSuffix(a.Val, ".htm") {
                        //fmt.Printf("-------------------------------\n")
                        //fmt.Printf(para.Data)
                        //fmt.Printf("-------------------------------\n")
                        return doc
                    }
                }

                if para != nil {
                    if curNode.FirstChild != nil {
                        ref, data := ProcessAncestorReference(curNode)
                        para.Data += curNode.FirstChild.Data
                        f := &Frag{data, ref, "", false}
                        para.Frags = append(para.Frags, f)
                    } else {
                        id := ProcessPersonIdentifier(curNode)
                        f := &Frag{"", "", id, false}
                        para.Frags = append(para.Frags, f)
                    }
                }
                curNode = curNode.NextSibling
            } else if curNode.Data == "b" {
                if para != nil {
                    //fmt.Printf("-------------------------------\n")
                    //fmt.Printf(para.Data)
                    //fmt.Printf("-------------------------------\n")
                    doc.Paragraphs = append(doc.Paragraphs, para)
                }
                // new paragraph
                para = new(Paragraph)
                para.Data += curNode.FirstChild.Data
                fmt.Printf("Name: %s\n", curNode.FirstChild.Data)
                f := &Frag{curNode.FirstChild.Data, "", "", false}
                para.Frags = append(para.Frags, f)

                curNode = curNode.NextSibling

            } else if curNode.Data == "p" {

                for sub := curNode.FirstChild; sub != nil; sub = sub.NextSibling {
                    if sub.Data == "a" {
                        para.Data += sub.FirstChild.Data

                        ref, data := ProcessAncestorReference(sub)
                        if ref == "" && data == "" {
                            doc.Paragraphs = append(doc.Paragraphs, para)
                            return doc
                        }
                        //fmt.Printf("Ref: %s %s\n", ref, data)
                        f := &Frag{data, ref, "", false}
                        para.Frags = append(para.Frags, f)

                    } else if sub.Data == "sup" {

                        for sub2 := sub.FirstChild; sub2 != nil; sub2 = sub2.NextSibling {
                            if sub2.Data == "a" {
                                var f *Frag
                                ref := sub2.FirstChild.Data
                                f = &Frag{"", ref, "", true}
                                para.Frags = append(para.Frags, f)
                                break
                            }
                        }
                    } else {
                        para.Data += sub.Data
                        f := &Frag{sub.Data, "", "", false}
                        para.Frags = append(para.Frags, f)
                    }
                }
                curNode = curNode.NextSibling
            } else if curNode.Data == "sup" {
                var f *Frag

                ref := curNode.FirstChild.Data

                for sub := curNode.FirstChild; sub != nil; sub = sub.NextSibling {
                    if sub.Data == "a" {
                        ref = sub.FirstChild.Data
                        f = &Frag{"", ref, "", true}
                        para.Frags = append(para.Frags, f)
                        break
                    }
                }

                curNode = curNode.NextSibling
            } else if curNode.Data == "hr" {
                curNode = curNode.NextSibling
            } else {
                // Encountered a tag we don't care about.  Increment the state
                curNode = curNode.NextSibling
            }
        } else {
            // handle the paragraph
            fmt.Printf(curNode.Data)
            if para != nil {
                para.Data += curNode.Data
                f := &Frag{curNode.Data, "", "", false}
                para.Frags = append(para.Frags, f)
            }
            curNode = curNode.NextSibling
        }
    }

    //fmt.Printf("-------------------------------\n")
    //fmt.Printf(para.Data)
    //fmt.Printf("-------------------------------\n")
    return doc
}

func Normalize(doc *Document) {
    for _, p := range(doc.Paragraphs) {
        for _, f := range(p.Frags) {
            var newF *Frag

            // normalize
            f.Data = strings.Replace(f.Data, "\n", " ", -1)
            f.RefId = strings.Replace(f.RefId, "\n", " ", -1)
            f.Identifier = strings.Replace(f.Identifier, "\n", " ", -1)

            if !f.IsSup {
                newF = new(Frag)
                for _, s := range(strings.Split(f.Data, "  ")) {
                    newS := strings.TrimSpace(s)
                    if newS != "" {
                        //fmt.Printf("(%s) ", newS)

                        newF.Data = newS
                        newF.RefId = f.RefId
                        newF.Identifier = f.Identifier
                        newF.IsSup = false

                        p.NormalizedFrags = append(p.NormalizedFrags, newF)
                    }

                    if strings.HasSuffix(newF.Data, ".") {
                        newF.Data = strings.TrimSuffix(newF.Data, ".")

                        newF = new(Frag)
                        newF.Data = "."
                        p.NormalizedFrags = append(p.NormalizedFrags, newF)
                    }

                    newF = new(Frag)
                }
            } else {
                newF = new(Frag)

                newF.Data = f.Data
                newF.RefId = f.RefId
                newF.Identifier = f.Identifier
                newF.IsSup = f.IsSup
                p.NormalizedFrags = append(p.NormalizedFrags, newF)
            }

            //fmt.Printf("\n")
        }
    }
}

func ProcessSentences(doc *Document) {
    for _, p := range(doc.Paragraphs) {
        var newSentence *Sentence

        newSentence = new(Sentence)

        for idx := 0; idx < len(p.NormalizedFrags); {

            newSentence.Frags = append(newSentence.Frags, p.NormalizedFrags[idx])

            if p.NormalizedFrags[idx].Data == "." {

                if idx + 1 < len(p.NormalizedFrags) {
                    if p.NormalizedFrags[idx + 1].IsSup {
                        newSentence.Frags =
                            append(newSentence.Frags, p.NormalizedFrags[idx + 1])
                        idx++
                    }
                }
                p.Sentences = append(p.Sentences, newSentence)
                newSentence = new(Sentence)
            }

            idx++
        }
        //fmt.Printf("\n")
    }
}

func ProcessBirth(s *Sentence, rec *Record) {
    nameWords := strings.Split(s.Frags[0].Data, " ")

    if len(nameWords) == 2 {
        rec.FirstName = nameWords[0]
        rec.LastName = nameWords[1]
    } else if len(nameWords) == 3 {
        rec.FirstName = nameWords[0]
        rec.MiddleName = nameWords[1]
        rec.LastName = nameWords[2]
    } else {
        rec.FirstName = nameWords[0]
    }

    rec.BirthDate = ProcessDatedEvent(s.AllWords())
}

func ProcessParents(s *Sentence, rec *Record) {

    idx := 0
    for _, f := range(s.Frags) {
        fmt.Printf("%s `%s` `%s`\n", f.Data, f.RefId, f.Identifier)
        if f.RefId != "" {
            p := &Parent{ Identifier : f.RefId, Name : f.Data }
            rec.Parents[idx] = p
            idx++
        }
    }
}

func ProcessChildren(s *Sentence, rec *Record) {
    for _, f := range(s.Frags) {
        if f.RefId != "" {
            fmt.Printf("Child: `%s`\n", f.Data)
            c := &Child { Identifier : f.RefId, Name : f.Data }
            rec.Children = append(rec.Children, c)
        }
    }
}

//func ProcessMarriage(s *Sentence, rec *Record) {
//    for _, f := range(s.Frags) {
//        if f.RefId != "" {
//            fmt.Printf("Married to: `%s`\n", f.Data)
//            m := &Marriage { OtherIdentifier : f.RefId, OtherName : f.Data }
//            rec.Marriages = append(rec.Marriages, m)
//            break
//        }
//    }
//
//    for _, f := range(s.Frags) {
//
//    }
//}

func ProcessCensus(s *Sentence, rec *Record) {

}

func ProcessOccupation(s *Sentence, rec *Record) {
}

func ProcessAlias(s *Sentence, rec *Record) {
}

func ProcessBurial(s *Sentence, rec *Record) {
}

func ProcessDeath(s *Sentence, rec *Record) {
}

func ProcessMarriageBond(s *Sentence, rec *Record) {
}

func ProcessResidence(s *Sentence, rec *Record) {
}

func ProcessDescription(s *Sentence, rec *Record) {
}

func GenerateRecords(doc *Document) []*Record {

    records := make([]*Record, 0)

    for _, p := range(doc.Paragraphs) {
        rec := NewRecord()

        for _, s := range(p.Sentences) {

            if s.Contains("was born") {
                ProcessBirth(s, rec)
            } else if s.Contains("appeared on the census") {
                //ProcessCensus(s, rec)
            } else if s.Contains("Parents:") {
                ProcessParents(s, rec)
            } else if s.Contains("Children were:") {
                ProcessChildren(s, rec)
            } else if s.Contains("was married to") {
                ProcessMarriage(s, rec)
            } else if s.Contains("was a") {
                //ProcessOccupation(s, rec)
            } else if s.Contains("also known as") {
                //ProcessAlias(s, rec)
            } else if s.Contains("was buried") {
                //ProcessBurial(s, rec)
            } else if s.Contains("died") {
                //ProcessDeath(s, rec)
            } else if s.Contains("was described as") {
                //ProcessDescription(s, rec)
            } else if s.Contains("listed as being born") {
                //ProcessBirthListing(s, rec)
            } else if s.Contains("date of marriage bond") {
                //ProcessMarriageBond(s, rec)
            } else if s.Contains("resided in") {
                //ProcessResidence(s, rec)
            } else {

                fmt.Printf("%s\n", s.String())
            }
        }
        records = append(records, rec)
    }

    return records
}

func IsDay(word string) (int, bool) {
    day, err := strconv.Atoi(word)

    if err != nil {
        return day, false
    }

    if day > MAX_MONTH_DAYS {
        return day, false
    }

    return day, true
}

func IsMonth(word string) (time.Month, bool) {
    month, ok := monthMap[word]

    if !ok {
        w := strings.TrimSuffix(word, ".")

        // We saw a string but it didn't resolve in our month map
        if _, err := strconv.Atoi(w); err != nil {
            log.Fatalf("Error: unexpected date element `%s`", w)
        }

        return time.January, false
    }

    return month, true
}

func IsYear(word string) (int, bool) {
    var year int
    var err error

    w := strings.TrimSuffix(word, ".")

    if year, err = strconv.Atoi(w); err != nil {
        return year, false
    }

    return year, true
}

func ParseCounty(words string) string {
    if !strings.HasSuffix(words, "Co.") {
        log.Printf("Warning: county `%s` doesn't end as expected",
                        words)
    }
    county := strings.TrimSuffix(words, "Co.")

    return strings.TrimSpace(county)
}

func ParseState(words string) string {
    //if !strings.HasSuffix(words, ".") &&
    //    !strings.HasSuffix(words, " . ") &&
    //    !strings.HasSuffix(words, " .  ") &&
    //    !strings.HasSuffix(words, " .") {

    //    log.Fatalf("Error: state `%s` doesn't end as expected",
    //                    words)
    //}
    state := strings.TrimSuffix(words, ".")

    return strings.TrimSpace(state)
}

func ProcessDatedEvent(words []string) *DatedEvent {
    var curPos int

    date := new(DatedEvent)

    for pos, w := range(words) {
        if (w == "on" || w == "in" || w == "about") {
            var ok bool

            curPos = pos + 1
            if date.d.day, ok = IsDay(words[curPos]); ok {
                curPos += 1
            }

            if date.d.month, ok = IsMonth(words[curPos]); ok {
                curPos += 1
            }

            if date.d.year, ok = IsYear(words[curPos]); ok {
                curPos += 1
            }

            if !ok {
                log.Fatalf("Error: expected date but date could not be parsed")
            }

            break
        }
    }

    if words[curPos] == "." {
        return date
    }

    if words[curPos] != "in" {
        log.Fatalf("Error: unexpected word `%s` found when processing location",
                        words[curPos])
    }

    curPos += 1

    // re-assemble end of sentence
    var locationPart string
    for pos, w := range(words[curPos:]) {
        locationPart += w

        // Add a space if its not the last token
        if pos != len(words[curPos:]) - 1 {
            locationPart += " "
        }
    }

    locToks := strings.Split(locationPart, ",")

    if len(locToks) > 3 {
        log.Fatalf("Error: heueristic will not match for `%s`", locationPart)
    }

    // If town, county and state are supplied
    if len(locToks) == 3 {
        date.loc.town = locToks[0]

        date.loc.county = ParseCounty(locToks[1])

        date.loc.state = ParseState(locToks[2])

    } else if len(locToks) == 2 {
        // County + State
        date.loc.county = ParseCounty(locToks[0])

        date.loc.state = ParseState(locToks[1])

    } else {
        // State
        //if !strings.HasSuffix(locToks[0], ".") {
        //    log.Fatalf("Error: state `%s` doesn't end as expected",
        //                    locToks[0])
        //}
        date.loc.state = ParseState(locToks[0])
    }
    return date
}

func main() {
    dirName := flag.String("d", "", "directory name")
    //mongoHost := flag.String("t", "", "mongo host")

    flag.Parse()

    //records := make(map[string]*Record, 0)
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

        doc, err := html.Parse(strings.NewReader(string(htmlText)))
        if err != nil {
            log.Fatal(err)
        }

        procDoc := ProcessDocument(doc)

        Normalize(procDoc)

        ProcessSentences(procDoc)

        //records := GenerateRecords(procDoc)
        GenerateRecords(procDoc)
    }

    // TODO: need a second pass to associate children with a marriage
}
