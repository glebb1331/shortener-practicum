package main

import (
	"fmt"
	"os"
)

func helper() {
	// os.Exit во вспомогательной функции — НЕ должен срабатывать.
	os.Exit(2)
}

func main() {
	fmt.Println("hello")
	os.Exit(0) // want `прямой вызов os\.Exit в функции main пакета main`
	os.Exit(1) // want `прямой вызов os\.Exit в функции main пакета main`
}
