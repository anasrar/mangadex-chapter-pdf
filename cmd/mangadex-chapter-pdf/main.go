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

type ChapterDataAttributes struct {
	Volume             string `json:"volume"`
	Chapter            string `json:"chapter"`
	Title              string `json:"title"`
	TranslatedLanguage string `json:"translatedLanguage"`
	ExternalUrl        string `json:"externalUrl"`
	PublishAt          string `json:"publishAt"`
	ReadableAt         string `json:"readableAt"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
	Pages              int    `json:"pages"`
	Version            int    `json:"version"`
}

type ChapterData struct {
	Id         string                `json:"id"`
	Type       string                `json:"type"`
	Attributes ChapterDataAttributes `json:"attributes"`
}

type Chapter struct {
	Result   string      `json:"result"`
	Response string      `json:"response"`
	Data     ChapterData `json:"data"`
}

type ChapterUrlImages struct {
	Hash      string   `json:"hash"`
	Data      []string `json:"data"`
	DataSaver []string `json:"dataSaver"`
}

type ChapterImages struct {
	Result  string           `json:"result"`
	BaseUrl string           `json:"baseUrl"`
	Chapter ChapterUrlImages `json:"chapter"`
}

var chapterInput = flag.String("chapter", "", "Url chapter, example: https://mangadex.org/chapter/xxxx-xxxx-xxxx-xxxx-xxxx\n")

func main() {
	flag.Parse()

	if *chapterInput == "" {
		log.Fatalln("-chapter can't be empty, -h to see example")
	}

	u, err := url.Parse(*chapterInput)
	if err != nil {
		log.Fatalf("ERROR PARSE URL CHAPTER: %s", err)
	}

	frag := strings.Split(u.Path, "/")
	if len(frag) < 3 {
		log.Fatalf("NOT VALID URL: %s", *chapterInput)
	}
	id := frag[2]

	log.Printf("GET CHAPTER DATA(id: %s)\n", id)
	resChapter, err := http.Get(fmt.Sprintf("https://api.mangadex.org/chapter/%s?includes%%5B%%5D=manga", id))
	if err != nil {
		log.Fatal(err)
	}
	dataChapter, err := io.ReadAll(resChapter.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("DONE")

	chapter := Chapter{}

	log.Printf("PARSE CHAPTER DATA(id: %s)\n", id)
	if err := json.Unmarshal(dataChapter, &chapter); err != nil {
		log.Fatal(err)
	}
	log.Println("DONE")

	log.Printf("GET CHAPTER URL IMAGES(id: %s)\n", id)
	resChapterImages, err := http.Get(fmt.Sprintf("https://api.mangadex.org/at-home/server/%s", id))
	if err != nil {
		log.Fatal(err)
	}
	dataChapterImages, err := io.ReadAll(resChapterImages.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("DONE")

	chapterImages := ChapterImages{}

	log.Printf("PARSE CHAPTER IMAGES(id: %s)\n", id)
	if err := json.Unmarshal(dataChapterImages, &chapterImages); err != nil {
		log.Fatal(err)
	}
	log.Println("DONE")

	images := make([]io.Reader, len(chapterImages.Chapter.Data))

	// NOTE: Possible to use routine for parallel download
	for i, url := range chapterImages.Chapter.Data {
		page := i + 1
		log.Printf("DOWNLOAD START(page: %d)\n", page)
		url := fmt.Sprintf("%s/data/%s/%s", chapterImages.BaseUrl, chapterImages.Chapter.Hash, url)
		res, err := http.Get(url)
		if err != nil {
			log.Printf("DOWNLOAD ERROR(page: %d): %s\n", page, err)
		} else {
			images[i] = bufio.NewReader(res.Body)
			log.Printf("DOWNLOAD DONE(page: %d)\n", page)
		}
	}

	out := &bytes.Buffer{}

	log.Println("CREATE PDF")
	if err := api.ImportImages(nil, out, images, nil, nil); err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(fmt.Sprintf("Chapter %s.pdf", chapter.Data.Attributes.Chapter), out.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
	log.Printf("SAVED PDF(file: Chapter %s.pdf)", chapter.Data.Attributes.Chapter)
	fmt.Println("Don't forget to support MangaDex")
}
