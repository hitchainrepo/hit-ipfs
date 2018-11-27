package main

import "C"
import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

//export RandomFunc
func RandomFunc(nodeId int) {
	rand.NewSource((int64(nodeId)))
	for i := 0; i < 10; i++{
		a := rand.Intn(100)
		fmt.Println(a)
	}
}

func stringToBin(s string) (binString string) {
	for _, c := range s {
		binString = fmt.Sprintf("%s%b",binString, c)
	}
	fmt.Println(binString)
	return binString
}

func BinDec(b string) (n int64) {
	s := strings.Split(b, "")
	l := len(s)
	i := 0
	d := float64(0)
	for i = 0; i < l; i++ {
		f, err := strconv.ParseFloat(s[i], 10)
		if err != nil {
			log.Println("Binary to decimal error:", err.Error())
			return -1
		}
		d += f * math.Pow(2, float64(l-i-1))
	}
	return int64(d)
}

func stringToInt64(s string) int64 {
	result := int64(0)
	for _, c := range s {
		result += int64(c)
	}
	return result
}

func main() {
	fmt.Print(stringToInt64("1231dwFEWR@#T"))
}
