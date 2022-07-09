package main

import (
	"fmt"
	"log"
	"math"
	"os"
)

const chunksize = 1 << (10)

func main() {
	filename := "file.txt"
	fi, err := os.Stat(filename) //使用fi.size得到文件大小
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(float64(fi.Size())) 查看大小
	file_num := math.Ceil(float64(fi.Size()) / float64(chunksize)) // 得到文件的分块数
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for i := 0; i < int(file_num); i++ {
		file_size := i * chunksize
		//off := int(fi.Size()) - (int(file_num)-1)*file_size
		buff := make([]byte, chunksize)
		_, _ = file.ReadAt(buff, int64(file_size))
		fmt.Println(".................")
		fmt.Println(string(buff))
	}

}
