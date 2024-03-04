package fileio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	lz4 "github.com/pierrec/lz4/v4"
)

type SaveMapHeader struct {
	Unknown1   [39]byte
	GameMode   uint8
	Difficulty uint8
}

type TileDataHeader struct {
	WorldCoordinates   [2]uint32
	Terrain            uint16
	Climate            uint16
	Altitude           int16
	Owner              uint8
	Capital            uint8
	CapitalCoordinates [2]int32
}

type TileData struct {
	Terrain  int
	Climate  int
	Owner    int
	Capital  int
	HasCity  bool
	CityName string
}

type PlayerData struct {
	Id                   int
	Name                 string
	AccountId            string
	AutoPlay             bool
	StartTileCoordinates [2]int
	Tribe                int
	Unknown              []byte
}

type UnitData struct {
	Unknown1           [4]byte
	Owner              uint8
	UnitType           uint16
	Unknown2           [8]byte
	CurrentCoordinates [2]uint32
	HomeCoordinates    [2]uint32
	Health             uint16 // should be divided by 10 to get value ingame
	PromotionLevel     uint16
	Experience         uint16
	Moved              bool
	Attacked           bool
	UnknownBool        bool
	CreatedTurn        uint16
}

type PolytopiaSaveOutput struct {
	MapHeight     int
	MapWidth      int
	OwnerTribeMap map[int]int
	TileData      [][]TileData
}

func readVarString(reader *io.SectionReader, varName string) string {
	variableLength := uint8(0)
	if err := binary.Read(reader, binary.LittleEndian, &variableLength); err != nil {
		log.Fatal("Failed to load variable length: ", err)
	}

	stringValue := make([]byte, variableLength)
	if err := binary.Read(reader, binary.LittleEndian, &stringValue); err != nil {
		log.Fatal(fmt.Sprintf("Failed to load string value. Variable length: %v, name: %s. Error:", variableLength, varName), err)
	}

	return string(stringValue[:])
}

func unsafeReadUint32(reader *io.SectionReader) uint32 {
	unsignedIntValue := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		log.Fatal("Failed to load uint32: ", err)
	}
	return unsignedIntValue
}

func unsafeReadInt32(reader *io.SectionReader) int32 {
	unsignedIntValue := int32(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		log.Fatal("Failed to load int32: ", err)
	}
	return unsignedIntValue
}

func unsafeReadUint16(reader *io.SectionReader) uint16 {
	unsignedIntValue := uint16(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		log.Fatal("Failed to load uint16: ", err)
	}
	return unsignedIntValue
}

func unsafeReadUint8(reader *io.SectionReader) uint8 {
	unsignedIntValue := uint8(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		log.Fatal("Failed to load uint8: ", err)
	}
	return unsignedIntValue
}

func buildReaderForDecompressedFile(inputFilename string) (*bytes.Reader, int) {
	inputFile, err := os.Open(inputFilename)
	defer inputFile.Close()
	if err != nil {
		log.Fatal("Failed to load state file: ", err)
		return nil, 0
	}

	inputBuffer := new(bytes.Buffer)
	inputBuffer.ReadFrom(inputFile)

	inputBytes := inputBuffer.Bytes()
	firstByte := inputBytes[0]
	sizeOfDiff := ((firstByte >> 6) & 3)
	if sizeOfDiff == 3 {
		sizeOfDiff = 4
	}
	dataOffset := 1 + int(sizeOfDiff)
	var resultDiff int
	if sizeOfDiff == 4 {
		resultDiff = int(binary.LittleEndian.Uint32(inputBytes[1 : 1+sizeOfDiff]))
	} else if sizeOfDiff == 2 {
		resultDiff = int(binary.LittleEndian.Uint16(inputBytes[1 : 1+sizeOfDiff]))
	} else {
		log.Fatal("Header sizeOfDiff is unrecognized value: ", sizeOfDiff)
	}
	dataLength := len(inputBytes) - dataOffset
	resultLength := dataLength + resultDiff

	fmt.Println("Result length:", resultLength)

	// decompress
	decompressedContents := make([]byte, resultLength)
	decompressedLength, err := lz4.UncompressBlock(inputBytes[dataOffset:], decompressedContents)
	if err != nil {
		panic(err)
	}

	fmt.Println("Decompressed data length:", decompressedLength)
	decompressedFilename := inputFilename + ".decomp"
	if err := os.WriteFile(decompressedFilename, decompressedContents[:decompressedLength], 0666); err != nil {
		log.Fatal("Error writing decompressed contents", err)
	}
	fmt.Println("Writing decompressed contents to", decompressedFilename)
	return bytes.NewReader(decompressedContents), decompressedLength
}

func readExistingCityData(streamReader *io.SectionReader, tileDataHeader TileDataHeader) TileData {
	cityLevel := unsafeReadUint32(streamReader)
	currentPopulation := unsafeReadUint16(streamReader)
	fmt.Println("City level:", cityLevel)
	fmt.Println("City current population:", currentPopulation)

	buffer1 := make([]byte, 12)
	if err := binary.Read(streamReader, binary.LittleEndian, &buffer1); err != nil {
		log.Fatal("Failed to load buffer: ", err)
	}

	cityName := readVarString(streamReader, "CityName")
	fmt.Println("Buffer1:", buffer1, ", cityName:", cityName)

	_ = unsafeReadUint8(streamReader) // discard zero
	unknownList1Size := unsafeReadUint16(streamReader)
	unknownList1 := make([]int, unknownList1Size+1)
	for i := 0; i < int(unknownList1Size)+1; i++ {
		unknownValue := unsafeReadUint16(streamReader)
		unknownList1[i] = int(unknownValue)
	}
	fmt.Println("unknownList1:", unknownList1)

	if unknownList1[len(unknownList1)-1] != 0 {
		// Seems related to rebellion
		buffer := make([]byte, 2)
		if err := binary.Read(streamReader, binary.LittleEndian, &buffer); err != nil {
			log.Fatal("Failed to load buffer: ", err)
		}
		fmt.Println("bufferRebel:", buffer)
	}

	unitFlag := unsafeReadUint8(streamReader)
	fmt.Println("Unit flag:", unitFlag)
	if unitFlag == 1 {
		fmt.Println("City has unit")
		unitData := UnitData{}
		if err := binary.Read(streamReader, binary.LittleEndian, &unitData); err != nil {
			log.Fatal("Failed to load buffer: ", err)
		}
		fmt.Println("Unit data:", unitData)

		flag1 := unsafeReadUint8(streamReader) // seems to always be zero
		flag2 := unsafeReadUint8(streamReader)
		bufferSize := 6
		if flag2 == 1 {
			bufferSize = 8
		}

		bufferUnit := make([]byte, bufferSize)
		if err := binary.Read(streamReader, binary.LittleEndian, &bufferUnit); err != nil {
			log.Fatal("Failed to load buffer: ", err)
		}
		fmt.Println("flag1:", flag1, ", flag2:", flag2, ", bufferUnit:", bufferUnit)
	}

	unknownList2Size := unsafeReadUint8(streamReader)
	unknownList2 := make([]byte, unknownList2Size+6)
	if err := binary.Read(streamReader, binary.LittleEndian, &unknownList2); err != nil {
		log.Fatal("Failed to load buffer: ", err)
	}
	fmt.Println("UnknownList2Size:", unknownList2Size, ", unknownList2:", unknownList2)

	return TileData{
		Terrain:  int(tileDataHeader.Terrain),
		Climate:  int(tileDataHeader.Climate),
		Owner:    int(tileDataHeader.Owner),
		Capital:  int(tileDataHeader.Capital),
		HasCity:  true,
		CityName: cityName,
	}
}

func readOtherTile(streamReader *io.SectionReader, tileDataHeader TileDataHeader, resourceType int, improvementType int) TileData {
	// Has improvement
	if improvementType != -1 {
		// Read improvement data
		improvementData := make([]byte, 23)
		if err := binary.Read(streamReader, binary.LittleEndian, &improvementData); err != nil {
			log.Fatal("Failed to load buffer: ", err)
		}
		fmt.Println("Remaining improvement data:", improvementData)
	}

	// Read unit data
	hasUnitFlag := unsafeReadUint8(streamReader)
	if hasUnitFlag == 1 { // unit flag
		fmt.Println("Tile has unit")
		unitData := UnitData{}
		if err := binary.Read(streamReader, binary.LittleEndian, &unitData); err != nil {
			log.Fatal("Failed to load buffer: ", err)
		}
		fmt.Println("Unit data:", unitData)

		hasOtherUnitFlag := unsafeReadUint8(streamReader)
		if hasOtherUnitFlag == 1 {
			fmt.Println("Tile has other unit data")
			unitData2 := UnitData{}
			if err := binary.Read(streamReader, binary.LittleEndian, &unitData2); err != nil {
				log.Fatal("Failed to load buffer: ", err)
			}
			fmt.Println("Unit data2:", unitData2)

			bufferUnitData2 := make([]byte, 15)
			if err := binary.Read(streamReader, binary.LittleEndian, &bufferUnitData2); err != nil {
				log.Fatal("Failed to load buffer: ", err)
			}
			fmt.Println("bufferUnitData2:", bufferUnitData2)
			if bufferUnitData2[1] == 1 {
				bufferUnitData3 := make([]byte, 4)
				if err := binary.Read(streamReader, binary.LittleEndian, &bufferUnitData3); err != nil {
					log.Fatal("Failed to load buffer: ", err)
				}
				fmt.Println("bufferUnitData3:", bufferUnitData3)
			}
		} else {
			bufferUnitFlag := unsafeReadUint8(streamReader)
			bufferSize := 6
			if bufferUnitFlag == 1 {
				bufferSize = 8
			}

			bufferUnit := make([]byte, bufferSize)
			if err := binary.Read(streamReader, binary.LittleEndian, &bufferUnit); err != nil {
				log.Fatal("Failed to load buffer: ", err)
			}
			fmt.Println("bufferUnit:", bufferUnit)
		}
	}

	unknownListSize := unsafeReadUint8(streamReader)
	fmt.Println("Read list with unknownListSize:", unknownListSize)
	unknownList := make([]byte, unknownListSize+6)
	if err := binary.Read(streamReader, binary.LittleEndian, &unknownList); err != nil {
		log.Fatal("Failed to load buffer: ", err)
	}
	fmt.Println("Unknown list data:", unknownList)

	hasCity := false
	if improvementType == 1 {
		hasCity = true // unexplored city, but has the improvement tile as city
	}

	return TileData{
		Terrain:  int(tileDataHeader.Terrain),
		Climate:  int(tileDataHeader.Climate),
		Owner:    int(tileDataHeader.Owner),
		Capital:  int(tileDataHeader.Capital),
		HasCity:  hasCity,
		CityName: "",
	}
}

func readTileData(streamReader *io.SectionReader, tileData [][]TileData, mapWidth int, mapHeight int) {
	for i := 0; i < int(mapWidth); i++ {
		for j := 0; j < int(mapHeight); j++ {
			tileDataHeader := TileDataHeader{}
			if err := binary.Read(streamReader, binary.LittleEndian, &tileDataHeader); err != nil {
				log.Fatal("Failed to load tileDataHeader: ", err)
			}
			fmt.Println(fmt.Sprintf("tileDataHeader (%v, %v): ", i, j, tileDataHeader))

			if int(tileDataHeader.WorldCoordinates[0]) != j || int(tileDataHeader.WorldCoordinates[1]) != i {
				log.Fatal(fmt.Sprintf("File reached unexpected location. Iteration (%v, %v) isn't equal to world coordinates (%v, %v)",
					i, j, tileDataHeader.WorldCoordinates[0], tileDataHeader.WorldCoordinates[1]))
			}

			resourceExistsFlag := unsafeReadUint8(streamReader)
			resourceType := -1
			if resourceExistsFlag == 1 {
				resourceType = int(unsafeReadUint16(streamReader))
				fmt.Println("Resource exists, type:", resourceType)
			}

			improvementExistsFlag := unsafeReadUint8(streamReader)
			improvementType := -1
			if improvementExistsFlag == 1 {
				improvementType = int(unsafeReadUint16(streamReader))
				fmt.Println("Improvement exists, type: ", improvementType)
			}

			// If tile is city, read differently
			if tileDataHeader.Owner > 0 && resourceType == -1 && improvementType == 1 {
				// No resource, but has improvement that is city
				tileData[i][j] = readExistingCityData(streamReader, tileDataHeader)
			} else {
				tileData[i][j] = readOtherTile(streamReader, tileDataHeader, resourceType, improvementType)
			}
		}
	}
}

func ReadPolytopiaSaveFile(inputFilename string) (*PolytopiaSaveOutput, error) {
	decompressedReader, decompressedLength := buildReaderForDecompressedFile(inputFilename)
	streamReader := io.NewSectionReader(decompressedReader, int64(0), int64(decompressedLength))

	mapHeader := SaveMapHeader{}
	if err := binary.Read(streamReader, binary.LittleEndian, &mapHeader); err != nil {
		return nil, err
	}
	fmt.Println("Map header:", mapHeader)

	mapName := readVarString(streamReader, "MapName")
	fmt.Println("Map name:", mapName)

	mapHeader2 := [69]byte{}
	if err := binary.Read(streamReader, binary.LittleEndian, &mapHeader2); err != nil {
		return nil, err
	}
	fmt.Println("Map header2:", mapHeader2)

	mapWidth := unsafeReadUint16(streamReader)
	mapHeight := unsafeReadUint16(streamReader)
	fmt.Println("Map width:", mapWidth, ", height:", mapHeight)

	tileData := make([][]TileData, mapHeight)
	for i := 0; i < int(mapHeight); i++ {
		tileData[i] = make([]TileData, mapWidth)
	}

	readTileData(streamReader, tileData, int(mapWidth), int(mapHeight))

	ownerTribeMap := make(map[int]int)
	numPlayers := unsafeReadUint16(streamReader)
	for i := 0; i < int(numPlayers); i++ {
		playerId := unsafeReadUint8(streamReader)
		playerName := readVarString(streamReader, "playerName")
		playerAccountId := readVarString(streamReader, "playerAccountId")
		autoPlay := unsafeReadUint8(streamReader)
		startTileCoordinates1 := unsafeReadInt32(streamReader)
		startTileCoordinates2 := unsafeReadInt32(streamReader)
		tribe := unsafeReadUint16(streamReader)

		buffer := make([]byte, 134)
		if err := binary.Read(streamReader, binary.LittleEndian, &buffer); err != nil {
			log.Fatal("Failed to load buffer: ", err)
		}

		playerData := PlayerData{
			Id:                   int(playerId),
			Name:                 playerName,
			AccountId:            playerAccountId,
			AutoPlay:             int(autoPlay) != 0,
			StartTileCoordinates: [2]int{int(startTileCoordinates1), int(startTileCoordinates2)},
			Tribe:                int(tribe),
			Unknown:              buffer,
		}

		mappedTribe, ok := ownerTribeMap[playerData.Id]
		if ok {
			log.Fatal(fmt.Sprintf("Owner to tribe map has duplicate player id %v already mapped to %v", playerData.Id, mappedTribe))
		}
		ownerTribeMap[playerData.Id] = playerData.Tribe
		fmt.Println("Player:", playerData)
	}
	fmt.Println("Owner to tribe map:", ownerTribeMap)

	partBetweenInitialAndCurrentMap := [44]byte{}
	if err := binary.Read(streamReader, binary.LittleEndian, &partBetweenInitialAndCurrentMap); err != nil {
		return nil, err
	}
	fmt.Println("partBetweenInitialAndCurrentMap:", partBetweenInitialAndCurrentMap)

	mapName2 := readVarString(streamReader, "MapName2")
	fmt.Println("Map name2:", mapName2)

	mapHeader3 := [69]byte{}
	if err := binary.Read(streamReader, binary.LittleEndian, &mapHeader3); err != nil {
		return nil, err
	}
	fmt.Println("Map header3:", mapHeader3)

	mapWidth2 := unsafeReadUint16(streamReader)
	mapHeight2 := unsafeReadUint16(streamReader)
	fmt.Println("Map width2:", mapWidth2, ", height2:", mapHeight2)

	readTileData(streamReader, tileData, int(mapWidth2), int(mapHeight2))

	output := &PolytopiaSaveOutput{
		MapHeight:     int(mapHeight),
		MapWidth:      int(mapWidth),
		OwnerTribeMap: ownerTribeMap,
		TileData:      tileData,
	}
	return output, nil
}
