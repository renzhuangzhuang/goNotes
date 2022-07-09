package main

import (
	"fmt"
	"log"
	"mmap"
)

func main() {
	at, _ := mmap.Open("file.txt")
	buff := make([]byte, 5)
	n, err := at.ReadAt(buff, 5)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(n)
	_ = at.Close()
	fmt.Println(string(buff))
}
