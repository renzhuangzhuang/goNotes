package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
)

func reduceName(mapTask int) string {
	return "mrtmp." + "-" + strconv.Itoa(mapTask) + ".json"
}
func main() {
	//第一步读取文件
	inputFileName := make([]*os.File, 12) // 这里后续都要改成 具体的任务数
	for i := 0; i < 12; i++ {
		file_name := reduceName(i)
		fmt.Println(file_name)
		inputFileName[i], _ = os.Open("D://go example//go 项目//MapReduce//map_function//" + file_name)
	}
	ans := make(map[string]int)
	for _, files := range inputFileName {
		result := make(map[string]int)
		dec := json.NewDecoder(files)
		dec.Decode(&result)
		for k, v := range result {
			ans[k] += v
		}
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
