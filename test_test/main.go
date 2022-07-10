package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func doMap(
	chunksize int, //缓存大小
	filename string, // 文件名称
	nReduceTask int, //reduce任务数，块数
	ans map[string]int, // 保存中间存放k-v
	wg *sync.WaitGroup,

) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	b1 := bufio.NewReader(file)
	for i := 0; i < nReduceTask; i++ {
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
		file_name := ReduceName(i)
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
	defer wg.Done()

}
func ReduceName(mapTask int) string {
	return "mrtmp." + "-" + strconv.Itoa(mapTask) + ".json"
}

func doReduce(
	filename string,
	result chan map[string]int,
	wg *sync.WaitGroup,
) {
	files, _ := os.Open(filename)
	ans := make(map[string]int)
	ans1 := make(map[string]int)
	dec := json.NewDecoder(files)
	dec.Decode(&ans)
	for k, v := range ans {
		ans1[k] += v
	}
	result <- ans1
	defer wg.Done()
}

const chunksize = 1 << (10) //二进制赋值

func main() {
	var wg sync.WaitGroup
	var wg1 sync.WaitGroup
	filename := "D:\\go example\\go 项目\\MapReduce\\map_function\\file.txt"
	ans := make(map[string]int)
	fi, err := os.Stat(filename) //使用fi.size得到文件大小
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(float64(fi.Size())) 查看大小
	file_num := math.Ceil(float64(fi.Size()) / float64(chunksize))
	nReduceTask := file_num
	wg.Add(1)
	go doMap(chunksize, filename, int(nReduceTask), ans, &wg)

	result := make([]chan map[string]int, int(nReduceTask))
	for i := 0; i < int(nReduceTask); i++ {
		result[i] = make(chan map[string]int, 10000)
	}
	for i := 0; i < int(nReduceTask); i++ {
		wg1.Add(1)
		file_name := ReduceName(i)
		file_name = "D://go example//go 项目//MapReduce//map_function//" + file_name
		go doReduce(file_name, result[i], &wg1)
	}
	wg.Wait()
	wg1.Wait()
	result_all := make(map[string]int)
	for _, v := range result {
		for k, value := range <-v {
			result_all[k] += value
		}
		if len(v) == 0 {
			close(v)
		}
	}
	sortmap := []string{}
	for k := range result_all {
		sortmap = append(sortmap, k)
	}
	sort.Strings(sortmap)
	//保存结果
	final_result, err := os.Create("result.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer final_result.Close()

	for _, v := range sortmap {
		final_result.WriteString(v + ":" + strconv.Itoa(result_all[v]) + "\n")
	}
}
