package main

import "C"
import (
	"fmt"
	"math/rand"
)

//export RandomFunc
func RandomFunc(nodeId int) {
	rand.NewSource((int64(nodeId)))
	for i := 0; i < 10; i++{
		a := rand.Intn(100)
		fmt.Println(a)
	}
}

func main() {
	RandomFunc(123)
}
