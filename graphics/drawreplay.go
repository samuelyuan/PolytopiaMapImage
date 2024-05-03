package graphics

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"log"
	"os"

	"github.com/samuelyuan/PolytopiaMapImage/fileio"
	"github.com/samuelyuan/PolytopiaMapImage/graphics/quantize"
)

const (
	GIF_DELAY = 100
)

var (
	drawMapColors = []color.Color{
		// terrain colors
		color.RGBA{95, 149, 149, 255},
		color.RGBA{47, 74, 93, 255},
		color.RGBA{105, 125, 54, 255},
		color.RGBA{238, 249, 255, 255},
		// tribe colors
		color.RGBA{0, 0, 0, 255},
		color.RGBA{54, 226, 170, 255},
		color.RGBA{243, 131, 129, 255},
		color.RGBA{53, 37, 20, 255},
		color.RGBA{255, 0, 153, 255},
		color.RGBA{153, 102, 0, 255},
		color.RGBA{0, 0, 255, 255},
		color.RGBA{0, 255, 0, 255},
		color.RGBA{171, 59, 214, 255},
		color.RGBA{255, 255, 0, 255},
		color.RGBA{39, 92, 74, 255},
		color.RGBA{255, 255, 255, 255},
		color.RGBA{204, 0, 0, 255},
		color.RGBA{125, 35, 28, 255},
		color.RGBA{255, 153, 0, 255},
		color.RGBA{182, 161, 133, 255},
		color.RGBA{194, 253, 0, 255},
		color.RGBA{128, 128, 128, 255},
		// mountain and forest colors
		color.RGBA{89, 90, 86, 255},
		color.RGBA{234, 244, 253, 255},
		color.RGBA{53, 72, 44, 255},
	}
)

type MapCoordinates struct {
	Coordinates [2]int
}

func buildCityToTerritoryMap(saveData *fileio.PolytopiaSaveOutput) map[string][]MapCoordinates {
	mapHeight := saveData.MapHeight
	mapWidth := saveData.MapWidth

	cityTerritoryMap := make(map[string][]MapCoordinates)

	for i := 0; i < mapHeight; i++ {
		for j := 0; j < mapWidth; j++ {
			tileData := saveData.TileData[i][j]
			capitalCoordinates := tileData.CapitalCoordinates
			cityKey := fmt.Sprintf("(%v,%v)", capitalCoordinates[0], capitalCoordinates[1])

			_, ok := cityTerritoryMap[cityKey]
			if !ok {
				cityTerritoryMap[cityKey] = make([]MapCoordinates, 0)
			}
			cityTerritoryMap[cityKey] = append(cityTerritoryMap[cityKey], MapCoordinates{Coordinates: [2]int{j, i}})
		}
	}

	return cityTerritoryMap
}

func captureCityTiles(
	saveData *fileio.PolytopiaSaveOutput,
	cityTerritoryMap map[string][]MapCoordinates,
	cityCoordinates0 int,
	cityCoordinates1 int,
	newPlayerId int,
) {
	cityKey := fmt.Sprintf("(%v,%v)", cityCoordinates0, cityCoordinates1)
	citySurroundingTiles := cityTerritoryMap[cityKey]
	for tileIndex := 0; tileIndex < len(citySurroundingTiles); tileIndex++ {
		tile := citySurroundingTiles[tileIndex]
		saveData.TileData[tile.Coordinates[1]][tile.Coordinates[0]].Owner = newPlayerId // int(captureEvent.PlayerId)
	}
}

func DrawReplay(saveData *fileio.PolytopiaSaveOutput, outputFilename string) {
	cityTerritoryMap := buildCityToTerritoryMap(saveData)

	// Build initial map from turn 1
	currentTileData := saveData.TileData
	saveData.TileData = saveData.InitialTileData

	// Assign territory around capitals to be consistent with current tile data
	for i := 0; i < saveData.MapHeight; i++ {
		for j := 0; j < saveData.MapWidth; j++ {
			tileData := saveData.TileData[i][j]

			if tileData.Capital > 0 {
				capitalCoordinates := tileData.CapitalCoordinates
				captureCityTiles(saveData, cityTerritoryMap, int(capitalCoordinates[0]), int(capitalCoordinates[1]), int(tileData.Capital))
			}
		}
	}

	outGif := &gif.GIF{}
	quantizer := quantize.MedianCutQuantizer{NumColor: 256}
	var mapPalette color.Palette
	mapPalette = drawMapColors

	for turn := 1; turn <= saveData.MaxTurn; turn++ {
		fmt.Println("Drawing frame for turn", turn)

		captureEvents := make([]fileio.ActionCaptureCity, 0)
		_, ok := saveData.TurnCaptureMap[turn]
		if ok {
			captureEvents = saveData.TurnCaptureMap[turn]
		}

		for eventNum := 0; eventNum < len(captureEvents); eventNum++ {
			captureEvent := captureEvents[eventNum]
			cityCoordinates0 := int(captureEvent.Coordinates[0])
			cityCoordinates1 := int(captureEvent.Coordinates[1])
			fmt.Println("Captured city at tile", captureEvent.Coordinates, "by player", int(captureEvent.PlayerId))

			// Assign city to new owner
			saveData.TileData[cityCoordinates1][cityCoordinates0].Owner = int(captureEvent.PlayerId)

			// If city hasn't been claimed by any player, assign the city name
			if saveData.TileData[cityCoordinates1][cityCoordinates0].ImprovementData != nil &&
				saveData.TileData[cityCoordinates1][cityCoordinates0].ImprovementData.CityName == "" {
				saveData.TileData[cityCoordinates1][cityCoordinates0].ImprovementData.CityName = currentTileData[cityCoordinates1][cityCoordinates0].ImprovementData.CityName
			}

			captureCityTiles(saveData, cityTerritoryMap, cityCoordinates0, cityCoordinates1, int(captureEvent.PlayerId))
		}

		mapImage := DrawMap(saveData)
		bounds := mapImage.Bounds()
		palettedImage := image.NewPaletted(bounds, nil)
		if mapPalette == nil {
			quantizer.Quantize(palettedImage, bounds, mapImage, image.ZP)
			mapPalette = palettedImage.Palette
		} else {
			quantizer.UseExistingPalette(palettedImage, bounds, mapImage, image.ZP, mapPalette)
		}

		outGif.Image = append(outGif.Image, palettedImage)
		outGif.Delay = append(outGif.Delay, GIF_DELAY)
	}

	outputFile, _ := os.OpenFile(outputFilename, os.O_WRONLY|os.O_CREATE, 0600)
	defer outputFile.Close()
	err := gif.EncodeAll(outputFile, outGif)
	if err != nil {
		log.Fatal("Error while saving GIF:", err)
	}
}
