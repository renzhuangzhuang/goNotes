package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"strconv"
)

func main() {
	//第一步读取文件

	inputFileName := "D://go example//go 项目//MapReduce//map_function//mrtmp.-0.json"
	files, err := os.Open(inputFileName)
	if err != nil {
		log.Fatal(err)
	}

	dec := json.NewDecoder(files)
	result := make(map[string]int)
	dec.Decode(&result)
	// 第二步 进行计数
	ans := make(map[string]int)
	for k, v := range result {
		ans[k] += v
	}
	// 第三部进行排序
	sortmap := []string{}
	for k := range ans {
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
		final_result.WriteString(v + ":" + strconv.Itoa(ans[v]) + "\n")
	}

}
