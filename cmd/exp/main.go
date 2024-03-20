package main

import (
	"fmt"
	"path/filepath"
)

func main() {

}

func images(imgs ...string) {
	fmt.Println(len(imgs))
}

func imageDir(dir string) string {
	imgDir := "images"
	return filepath.Join(imgDir, dir)
}
