package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

type Chapter struct {
	Hash      string   `json:"hash"`
	Data      []string `json:"data"`
	DataSaver []string `json:"dataSaver"`
}

type ChapterImages struct {
	Result  string  `json:"ok"`
	BaseUrl string  `json:"baseUrl"`
	Chapter Chapter `json:"chapter"`
}

var chapter = flag.String("chapter", "", "Url chapter, example: https://mangadex.org/chapter/xxxx-xxxx-xxxx-xxxx-xxxx\n")

func main() {
	flag.Parse()

	if *chapter == "" {
		log.Fatalln("-chapter can't be empty, -h to see example")
	}

	u, err := url.Parse(*chapter)
	if err != nil {
		log.Fatalf("ERROR PARSE URL CHAPTER: %s", err)
	}

	frag := strings.Split(u.Path, "/")
	if len(frag) < 3 {
		log.Fatalf("NOT VALID URL: %s", *chapter)
	}

	id := frag[2]
	log.Printf("GET CHAPTER URL IMAGES(id: %s)\n", id)
	resChapterImages, err := http.Get(fmt.Sprintf("https://api.mangadex.org/at-home/server/%s", id))
	if err != nil {
		log.Fatal(err)
	}
	data, err := io.ReadAll(resChapterImages.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("DONE")

	info := ChapterImages{}

	log.Printf("PARSE CHAPTER(id: %s)\n", id)
	if err := json.Unmarshal(data, &info); err != nil {
		log.Fatal(err)
	}
	log.Println("DONE")

	images := make([]io.Reader, len(info.Chapter.Data))

	// NOTE: Possible to use routine for parallel download
	for i, url := range info.Chapter.Data {
		log.Printf("DOWNLOAD START(page: %d)\n", i)
		url := fmt.Sprintf("%s/data/%s/%s", info.BaseUrl, info.Chapter.Hash, url)
		res, err := http.Get(url)
		if err != nil {
			log.Printf("DOWNLOAD ERROR(page: %d): %s\n", i, err)
		} else {
			images[i] = bufio.NewReader(res.Body)
			log.Printf("DOWNLOAD DONE(page: %d)\n", i)
		}
	}

	out := &bytes.Buffer{}

	log.Println("CREATE PDF")
	if err := api.ImportImages(nil, out, images, nil, nil); err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s.pdf", info.Chapter.Hash), out.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
	log.Printf("SAVED PDF(file: %s.pdf)", info.Chapter.Hash)
	fmt.Println("Don't forget to support MangaDex")
}
