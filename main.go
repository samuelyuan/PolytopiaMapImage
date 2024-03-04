package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/samuelyuan/PolytopiaMapImage/fileio"
	"github.com/samuelyuan/PolytopiaMapImage/graphics"
)

func main() {
	inputPtr := flag.String("input", "", "Input filename")
	outputPtr := flag.String("output", "output.png", "Output filename")

	flag.Parse()

	inputFilename := *inputPtr
	outputFilename := *outputPtr
	fmt.Println("Input filename: ", inputFilename)
	fmt.Println("Output filename: ", outputFilename)

	saveFileData, err := fileio.ReadPolytopiaSaveFile(inputFilename)
	if err != nil {
		log.Fatal("Failed to load save file: ", err)
		return
	}

	graphics.SaveImage(outputFilename, graphics.DrawMap(saveFileData))
}
