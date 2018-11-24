package main

import (
	"container/list"
	"fmt"
	"math/rand"
	"C"
)

func randomFunc() *list.List {
	nodeId := 123
	rand.NewSource((int64(nodeId)))
	result := list.New()
	for i := 0; i < 10; i++{
		a := rand.Intn(100)
		fmt.Println(a)
		result.PushBack(a)
	}
	return result
}

func test(){
	var s int
	for a := 0; a <= 1000000; a++ {
		s += a
	}
	fmt.Println(s)
}

func main(){

}
