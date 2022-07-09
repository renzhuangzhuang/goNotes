package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

const chunksize = 1 << (10) //二进制赋值
func reduceName(mapTask int) string {
	return "mrtmp." + "-" + strconv.Itoa(mapTask) + ".json"
}

func main() {
	filename := "file.txt"
	ans := make(map[string]int)
	fi, err := os.Stat(filename) //使用fi.size得到文件大小
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(float64(fi.Size())) 查看大小
	file_num := math.Ceil(float64(fi.Size()) / float64(chunksize)) // 得到文件的分块数
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	b1 := bufio.NewReader(file)

	for i := 0; i < int(file_num); i++ {
		p := make([]byte, chunksize)
		n1, err := b1.Read(p)
		if err != nil {
			log.Fatal(err)
		}

		ss := strings.Fields(string(p[:n1]))

		for _, v := range ss {

			word := strings.ToLower(v)
			for len(word) > 0 && (word[0] < 'a' || word[0] > 'z') {
				word = word[1:]
			}
			for len(word) > 0 && (word[len(word)-1] < 'a' || word[len(word)-1] > 'z') {
				word = word[:len(word)-1]
			}
			ans[word]++
		}
		/* b, err := json.Marshal(ans)
		if err != nil {
			log.Fatal(err)
		}
		file_name := reduceName(i)
		ioutil.WriteFile(file_name, b, os.ModeAppend) */
		file_name := reduceName(i)
		files, err := os.Create(file_name)
		if err != nil {
			panic(err)
		}
		taskJson, err := json.Marshal(ans)
		if err != nil {
			panic(err)
		}
		if _, err := files.Write(taskJson); err != nil {
			panic(err)
		}
		if err := files.Close(); err != nil {
			panic(err)
		}

	}
}
