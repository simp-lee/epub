package epub_test

import (
	"fmt"
	"log"

	"github.com/simp-lee/epub"
)

func ExampleOpen() {
	book, err := epub.Open("testdata/book.epub")
	if err != nil {
		log.Fatal(err)
	}
	defer book.Close()

	md := book.Metadata()
	fmt.Println(md.Titles[0])
}

func ExampleNewReader() {
	// NewReader works with any io.ReaderAt, such as an *os.File or bytes.Reader.
	// f, _ := os.Open("book.epub")
	// info, _ := f.Stat()
	// book, err := epub.NewReader(f, info.Size())

	_ = epub.NewReader // placeholder — see Open example for full usage
}

func ExampleBook_Metadata() {
	book, err := epub.Open("testdata/book.epub")
	if err != nil {
		log.Fatal(err)
	}
	defer book.Close()

	md := book.Metadata()

	fmt.Printf("Title:   %s\n", md.Titles[0])
	fmt.Printf("Version: %s\n", md.Version)

	for _, a := range md.Authors {
		fmt.Printf("Author:  %s\n", a.Name)
	}
}

func ExampleBook_TOC() {
	book, err := epub.Open("testdata/book.epub")
	if err != nil {
		log.Fatal(err)
	}
	defer book.Close()

	for _, item := range book.TOC() {
		fmt.Printf("%s → %s\n", item.Title, item.Href)
	}
}

func ExampleBook_Chapters() {
	book, err := epub.Open("testdata/book.epub")
	if err != nil {
		log.Fatal(err)
	}
	defer book.Close()

	for _, ch := range book.Chapters() {
		text, err := ch.TextContent()
		if err != nil {
			continue
		}
		fmt.Printf("%-20s %d chars\n", ch.Title, len(text))
	}
}

func ExampleBook_Cover() {
	book, err := epub.Open("testdata/book.epub")
	if err != nil {
		log.Fatal(err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		fmt.Println("no cover found")
		return
	}

	fmt.Printf("Cover: %s (%s, %d bytes)\n", cover.Path, cover.MediaType, len(cover.Data))
}
