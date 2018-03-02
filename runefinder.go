package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/atotto/clipboard"
	homedir "github.com/mitchellh/go-homedir"
)

const ucdFileName = "UnicodeData.txt"
const ucdBaseUrl = "http://www.unicode.org/Public/UCD/latest/ucd/"

const runefinderHome = ".runefinder"

var baseDir = path.Join(getHome(), runefinderHome)

type runefinder struct {
	config Config
	index  map[string][]rune
	names  map[rune]string
}

// Config contains all configuration options
type Config struct {
	partialSearch bool
	regexSearch   bool
	displayHelp   bool
	update        bool
	query         string
}

func progressDisplay(running <-chan bool) {
	for {
		select {
		case <-running:
			fmt.Println("xxx")
		case <-time.After(200 * time.Millisecond):
			fmt.Print(".")
		}
	}
}

func getUcdFile(fileName string) {

	url := ucdBaseUrl + ucdFileName
	fmt.Printf("retrieving %s from %s\n", ucdFileName, url)
	/*running := make(chan bool)
	go progressDisplay(running)
	defer func() {
		running <- false
	}()
	*/
	response, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(file, response.Body)
	if err != nil {
		panic(err)
	}
	file.Close()
}

func (app *runefinder) buildIndex(fileName string) {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		getUcdFile(fileName)
	}

	if app.config.update {
		getUcdFile(fileName)
		return
	}

	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(content), "\n")

	index := map[string][]rune{}
	names := map[rune]string{}

	for _, line := range lines {
		var uchar rune
		fields := strings.Split(line, ";")
		if len(fields) >= 2 {
			code64, _ := strconv.ParseInt(fields[0], 16, 0)
			uchar = rune(code64)
			names[uchar] = fields[1]
			// fmt.Printf("%#v", index)
			for _, word := range strings.Split(fields[1], " ") {
				var entries []rune
				if len(index[word]) < 1 {
					entries = make([]rune, 0)
				} else {
					entries = index[word]
				}
				index[word] = append(entries, uchar)
			}
		}

	}
	app.index = index
	app.names = names
}

func (app *runefinder) findRunes(query string) []rune {
	found := []rune{}

	query = strings.ToUpper(query)

	if app.config.partialSearch {
		for key, _ := range app.index {
			if strings.Contains(key, query) {
				for _, uchar := range app.index[key] {
					found = append(found, uchar)
				}
			}
		}
		return found
	}

	if app.config.regexSearch {

		reggie := regexp.MustCompile(query)
		for idx, _ := range app.names {
			if reggie.MatchString(app.names[idx]) {
				found = append(found, idx)
			}
		}

		return found
	}

	for _, uchar := range app.index[query] {
		found = append(found, uchar)
	}
	return found
}

// Get user home directory or exit with a fatal error.
func getHome() string {

	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}

	return homeDir
}

func main() {

	var app runefinder

	flag.BoolVarP(&app.config.partialSearch, "partial", "p", false, "run a partial word search")
	flag.BoolVarP(&app.config.regexSearch, "regex", "r", false, "run a regex search")
	flag.BoolVar(&app.config.update, "update", false, "update the unicode database")
	flag.BoolVarP(&app.config.displayHelp, "help", "h", false, "display help")
	flag.Parse()

	// override the default usage display
	if app.config.displayHelp {
		displayUsage()
		os.Exit(0)
	}

	if len(flag.Args()) != 1 && !app.config.update {
		displayUsage()
		os.Exit(1)
	}
	word := flag.Arg(0)

	//create our runefinder directory if it doesn't exist
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		os.Mkdir(baseDir, 0755)
	}

	path := path.Join(baseDir, ucdFileName)
	app.buildIndex(path)

	if app.config.update {
		os.Exit(0)
	}

	count := 0
	format := "U+%04X  %c \t%s\n"
	var lastCharFound rune
	for _, uchar := range app.findRunes(word) {
		if uchar > 0xFFFF {
			format = "U+%5X %c \t%s\n"
		}
		lastCharFound = uchar
		fmt.Printf(format, uchar, uchar, app.names[uchar])
		count++
	}

	//if only one character was found, copy the character to clipboard
	if count == 1 {
		clipboard.WriteAll(string(lastCharFound))
		fmt.Printf("%s copied to the clipboard!\n", string(lastCharFound))
	} else {
		fmt.Printf("%d characters found\n", count)
	}
}

// print custom usage instead of the default provided by pflag
func displayUsage() {

	fmt.Printf("Usage: runefinder [<flags>] <word>\n\n")
	fmt.Printf("Example: runefinder cat\n\n")
	fmt.Printf("Optional flags:\n\n")
	flag.PrintDefaults()
}
