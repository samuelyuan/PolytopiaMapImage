package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/samuelyuan/PolytopiaMapImage/graphics"
	polytopiamapmodel "github.com/samuelyuan/polytopiamapmodelgo"
)

func main() {
	inputPtr := flag.String("input", "", "Input filename")
	outputPtr := flag.String("output", "output.png", "Output filename")
	modePtr := flag.String("mode", "image", "Output mode")

	flag.Parse()

	inputFilename := *inputPtr
	outputFilename := *outputPtr
	mode := *modePtr
	fmt.Println("Input filename: ", inputFilename)
	fmt.Println("Output filename: ", outputFilename)
	fmt.Println("Mode:", mode)

	saveFileData, err := polytopiamapmodel.ReadPolytopiaCompressedFile(inputFilename)
	if err != nil {
		log.Fatal("Failed to load save file: ", err)
		return
	}

	if mode == "image" {
		graphics.SaveImage(outputFilename, graphics.DrawMap(saveFileData))
	} else if mode == "replay" {
		graphics.DrawReplay(saveFileData, outputFilename)
	} else {
		log.Fatal("Invalid mode:", mode)
	}
}
