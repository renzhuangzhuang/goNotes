package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

const chunkSize = 10

func main() {
	// open file
	f, err := os.Open("file.txt")
	if err != nil {
		log.Fatal(err)
	}
	// remember to close the file at the end of the program
	defer f.Close()

	buf := make([]byte, chunkSize)

	for {
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		if err == io.EOF {
			break
		}

		fmt.Println(string(buf[:n]))
	}
}
