package main

import (
	"encoding/csv"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"os"
	"runtime"
	"sort"
	"strconv"
)

var osType = runtime.GOOS

func GetProjectPath() string {
	var projectPath string
	projectPath, _ = os.Getwd()
	return projectPath
}

func main() {
	// Window系统路径，其他系统注意修改路径
	// 此处使用的是全局路径
	f, err := os.Open("D:\\my\\GoML\\Go语言机器学习实战\\source\\02\\train.csv")
	mHandleErr(err)
	hdr, data, indices, err := ingest(f)
	mHandleErr(err)
	c := cardinality(indices)

	fmt.Printf("Original Data: \nRows: %d, Cols: %d\n========\n", len(data), len(hdr))
	//c := cardinality(indices)
	for i, h := range hdr {
		fmt.Printf("%v: %v\n", h, c[i])
	}
	fmt.Println()
}

func ingest(f io.Reader) (header []string, data [][]string, indices []map[string][]int, err error) {
	r := csv.NewReader(f)
	if header, err = r.Read(); err != nil {
		return
	}
	indices = make([]map[string][]int, len(header))
	var rowCount, colCount int = 0, len(header)
	for rec, err := r.Read(); err == nil; rec, err = r.Read() {
		if len(rec) != colCount {
			return nil, nil, nil,
				errors.Errorf("Expected Columns: %d. Got %d columns in row %d", colCount, len(rec), rowCount)
		}
		data = append(data, rec)
		for j, val := range rec {
			if indices[j] == nil {
				indices[j] = make(map[string][]int)
			}
			indices[j][val] = append(indices[j][val], rowCount)
		}
		rowCount++
	}
	return
}

func cardinality(indices []map[string][]int) []int {
	retVal := make([]int, len(indices))
	for i, m := range indices {
		retVal[i] = len(m)
	}
	return retVal
}

func mHandleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// 提示信息是一个用于表明是否为分类变量的布尔值切片
func clean(hdr []string, data [][]string, indices []map[string][]int, hints []bool, ignored []string) (int, int, []float64, []float64, []string, []bool) {
	modes := mode(indices)
	var Xs, Ys []float64
	var newHints []bool
	var newHdr []string
	var cols int

	for i, row := range data {
		for j, col := range row {
			if hdr[j] == "Id" {
				continue
			}
			if hdr[j] == "SalePrice" {
				cxx, _ := convert(col, false, nil, hdr[j])
				Ys = append(Ys, cxx...)
				continue
			}
			if inList(hdr[i], ignored) {
				continue
			}
			if hints[j] {
				col = imputeCategorical(col, j, hdr, modes)
			}
			cxx, newHdrs := convert(col, hints[j], indices[j], hdr[j])
			Xs = append(Xs, cxx...)
			if i == 0 {
				h := make([]bool, len(cxx))
				for k := range h {
					h[k] = hints[j]
				}
				newHints = append(newHints, h...)
				newHdr = append(newHdr, newHdrs...)
			}
		}
		if i == 0 {
			cols = len(Xs)
		}
	}
	rows := len(data)
	if len(Ys) == 0 {
		Ys = make([]float64, len(data))
	}
	return rows, cols, Xs, Ys, newHdr, newHints
}

// imputeCategorical函数是利用分类值模式来替换"NA"
func imputeCategorical(a string, col int, hdr []string, modes []string) string {
	if a != "NA" || a != "" {
		return a
	}
	switch hdr[col] {
	case "MSZoning", "BsmtFullBath", "BsmtHalfBath", "Utilities", "Functional",
		"Electrical", "KitchenQual", "SaleType", "Exterior1st", "Exterior2nd":
		return modes[col]
	}
	return a
}

// 模式是对每个变量查找最常见的值
func mode(index []map[string][]int) []string {
	retVal := make([]string, len(index))
	for i, m := range index {
		var max int
		for k, v := range m {
			if len(v) > max {
				max = len(v)
				retVal[i] = k
			}
		}
	}
	return retVal
}

// convert函数是将字符串转换为浮点型切片
func convert(a string, isCat bool, index map[string][]int, varName string) ([]float64, []string) {
	if isCat {
		return convertCategorical(a, index, varName)
	}
	f, _ := strconv.ParseFloat(a, 64)
	return []float64{f}, []string{varName}
}

// 讲分类变量编码为一个浮点型切片
func convertCategorical(a string, index map[string][]int, varName string) ([]float64, []string) {
	retVal := make([]float64, len(index)-1)
	temp := make([]string, 0, len(index))
	for k := range index {
		temp = append(temp, k)
	}
	temp = tryNumCat(a, index, temp)

	var naIndex int
	for i, v := range temp {
		if v == "NA" {
			naIndex = i
			break
		}
	}
	temp[0], temp[naIndex] = temp[naIndex], temp[0]
	for i, v := range temp[1:] {
		if v == a {
			retVal[i] = 1
			break
		}
	}
	for i, v := range temp {
		temp[i] = fmt.Sprintf("%v_%v", varName, v)
	}
	return retVal, temp[1:]
}

func tryNumCat(a string, index map[string][]int, catStrs []string) []string {
	isNumCat := true
	cats := make([]int, 0, len(index))
	for k := range index {
		i64, err := strconv.ParseInt(k, 10, 64)
		if err != nil && k != "NA" {
			isNumCat = false
			break
		}
		cats = append(cats, int(i64))
	}

	if isNumCat {
		sort.Ints(cats)
		for i := range cats {
			catStrs[i] = strconv.Itoa(cats[i])
		}
		if _, ok := index["NA"]; ok {
			catStrs[0] = "NA" // there are no negative numerical categories
		}
	} else {
		sort.Strings(catStrs)
	}
	return catStrs
}

func inList(a string, l []string) bool {
	for _, v := range l {
		if a == v {
			return true
		}
	}
	return false
}
