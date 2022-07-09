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
	"sync"
	"unicode"
)

type KeyValue struct {
	key   string
	value string
}

const chunksize = 1 << (10) //二进制赋值

//生成文件名称
func reduceName(mapTask int) string {
	return "mrtmp." + "-" + strconv.Itoa(mapTask) + ".txt"
}

func mapF(contents string) []KeyValue {
	//debug("Map %v\n", value)
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	keys := strings.FieldsFunc(contents, f)
	var res []KeyValue
	for _, key := range keys {
		res = append(res, KeyValue{key, "1"})
	}
	return res

}

func main() {
	var wg sync.WaitGroup
	filename := "file.txt"
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
	ans := make([][]KeyValue, int(file_num))
	b1 := bufio.NewReader(file)
	for i := 0; i < int(file_num); i++ {
		p := make([]byte, chunksize)
		b1.Read(p)
		wg.Add(1)
		go func(b []byte, a int) {
			defer wg.Done()
			res1 := mapF(string(b))
			for _, keyValuePair := range res1 {
				ans[a] = append(ans[a], keyValuePair)
			}
		}(p, i)
	}
	wg.Wait()

	for index, task := range ans {

		//file_name := reduceName(index)
		fmt.Println(index)
		b, e := json.Marshal(task)
		fmt.Println(e)
		fmt.Println(b)
		fmt.Println(string(b))
	}

}
