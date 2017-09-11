package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	homedir "github.com/mitchellh/go-homedir"
)

const ucdFileName = "UnicodeData.txt"
const ucdBaseUrl = "http://www.unicode.org/Public/UCD/latest/ucd/"

const runefinderHome = ".runefinder"

var baseDir = path.Join(getHome(), runefinderHome)

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
	fmt.Printf("%s not found\nretrieving from %s\n", ucdFileName, url)
	running := make(chan bool)
	go progressDisplay(running)
	defer func() {
		running <- false
	}()
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

func buildIndex(fileName string) (map[string][]rune, map[rune]string) {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		getUcdFile(fileName)
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
	return index, names
}

func findRunes(query string, index map[string][]rune) []rune {
	found := []rune{}
	for _, uchar := range index[strings.ToUpper(query)] {
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
	if len(os.Args) != 2 {
		fmt.Println("Usage:  runefinder <word>\texample: runefinder cat")
		os.Exit(1)
	}
	word := os.Args[1]

	//create our app directory if it doesn't exist
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		os.Mkdir(baseDir, 0755)
	}

	path := path.Join(baseDir, ucdFileName)
	index, names := buildIndex(path)

	count := 0
	format := "U+%04X  %c \t%s\n"
	var lastCharFound rune
	for _, uchar := range findRunes(word, index) {
		if uchar > 0xFFFF {
			format = "U+%5X %c \t%s\n"
		}
		lastCharFound = uchar
		fmt.Printf(format, uchar, uchar, names[uchar])
		count++
	}
	fmt.Printf("%d characters found\n", count)

	//if only one character was found, copy the character to clipboard
	if count == 1 {
		clipboard.WriteAll(string(lastCharFound))
	}
}
