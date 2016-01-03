package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"github.com/BlackEspresso/htmlcheck"
	"github.com/BlackEspresso/crawlbase"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func main() {
	var url, inputPath string
	flag.StringVar(&url, "url", "http://google.com", "http://google.com")
	flag.StringVar(&inputPath, "path", "./files/", "output path")
	flag.Parse()

	validater := htmlcheck.Validator{}
	tags := loadTagsFromFile()
	validater.AddValidTags(tags)
	// first check
	files, err := ioutil.ReadDir(inputPath)
	logFatal(err)
	records := [][]string{{"url", "error"}}

	for _, k := range files {
		fraw, err := ioutil.ReadFile(path.Join(inputPath, k.Name()))
		logFatal(err)
		var page crawlbase.Page
		err = json.Unmarshal(fraw, &page)
		logFatal(err)
		errors := validater.ValidateHtmlString(page.Body)
		ioutil.WriteFile("out.html",[]byte(page.Body),755)
		text := ""
		for _,k := range errors {
			text += k.Error() + "\n"
		}
		row := []string{page.Url, text}
		records = append(records, row)
	}
	toCSV(records)
}

func toCSV(records [][]string) {
	f, err := os.Create("urls.csv")
	defer f.Close()
	logFatal(err)

	w := csv.NewWriter(f)
	w.WriteAll(records) // calls Flush internally

	if err := w.Error(); err != nil {
		log.Fatalln("error writing csv:", err)
	}
}

func loadTagsFromFile() []htmlcheck.ValidTag {
	content, err := ioutil.ReadFile("tags.json")
	logFatal(err)

	var validTags []htmlcheck.ValidTag
	err = json.Unmarshal(content, &validTags)
	logFatal(err)

	return validTags
}

func logFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
