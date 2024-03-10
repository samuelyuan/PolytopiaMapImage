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

type MapHeaderInput struct {
	Version1     uint32
	Version2     uint32
	Unknown1     [2]byte
	CurrentTurn  uint16
	Unknown2     [3]byte
	MaxUnitId    uint32
	UnknownByte1 uint8
	Seed         uint32
	TurnLimit    uint32
	Unknown3     [11]byte
	GameMode1    uint8
	GameMode2    uint8
}

type MapHeaderOutput struct {
	MapName           string
	MapSquareSize     int
	DisabledTribesArr []int
	UnlockedTribesArr []int
	GameDifficulty    int
	NumOpponents      int
	UnknownArr        []byte
	SelectedTribes    map[int]int
	MapWidth          int
	MapHeight         int
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
	Unit     *UnitData
}

type PlayerData struct {
	Id                   int
	Name                 string
	AccountId            string
	AutoPlay             bool
	StartTileCoordinates [2]int
	Tribe                int
	UnknownByte1         int
	UnknownInt1          int
	UnknownArr1          []int
	Currency             int
	Score                int
	UnknownInt2          int
	NumCities            int
	AvailableTech        []int
	EncounteredPlayers   []int
	Tasks                []PlayerTaskData
	TotalUnitsKilled     int
	TotalUnitsLost       int
	TotalTribesDestroyed int
	UnknownBuffer1       []byte
	UniqueImprovements   []int
	DiplomacyArr         []DiplomacyData
	DiplomacyMessages    []DiplomacyMessage
	DestroyedByTribe     int
	DestroyedTurn        int
	UnknownBuffer2       []byte
}

type UnitData struct {
	Id                 uint32
	Owner              uint8
	UnitType           uint16
	Unknown            [8]byte // seems to be all zeros
	CurrentCoordinates [2]int32
	HomeCoordinates    [2]int32
	Health             uint16 // should be divided by 10 to get value ingame
	PromotionLevel     uint16
	Experience         uint16
	Moved              bool
	Attacked           bool
	Flipped            bool
	CreatedTurn        uint16
}

type ImprovementData struct {
	Level     uint16
	Founded   uint16
	Unknown1  [6]byte
	BaseScore uint16
	Unknown2  [11]byte
}

type PlayerTaskData struct {
	Type   int
	Buffer []byte
}

type DiplomacyMessage struct {
	MessageType int
	Sender      int
}

type DiplomacyData struct {
	PlayerId               uint8
	DiplomacyRelationState uint8
	LastAttackTurn         int32
	EmbassyLevel           uint8
	LastPeaceBrokenTurn    int32
	FirstMeet              int32
	EmbassyBuildTurn       int32
	PreviousAttackTurn     int32
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
	signedIntValue := int32(0)
	if err := binary.Read(reader, binary.LittleEndian, &signedIntValue); err != nil {
		log.Fatal("Failed to load int32: ", err)
	}
	return signedIntValue
}

func unsafeReadUint16(reader *io.SectionReader) uint16 {
	unsignedIntValue := uint16(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		log.Fatal("Failed to load uint16: ", err)
	}
	return unsignedIntValue
}

func unsafeReadInt16(reader *io.SectionReader) int16 {
	signedIntValue := int16(0)
	if err := binary.Read(reader, binary.LittleEndian, &signedIntValue); err != nil {
		log.Fatal("Failed to load int16: ", err)
	}
	return signedIntValue
}

func unsafeReadUint8(reader *io.SectionReader) uint8 {
	unsignedIntValue := uint8(0)
	if err := binary.Read(reader, binary.LittleEndian, &unsignedIntValue); err != nil {
		log.Fatal("Failed to load uint8: ", err)
	}
	return unsignedIntValue
}

func readFixedList(streamReader *io.SectionReader, listSize int) []byte {
	buffer := make([]byte, listSize)
	if err := binary.Read(streamReader, binary.LittleEndian, &buffer); err != nil {
		log.Fatal("Failed to load buffer: ", err)
	}
	return buffer
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
	currentPopulation := unsafeReadInt16(streamReader)
	totalPopulation := unsafeReadUint16(streamReader)
	fmt.Println("City level:", cityLevel)
	fmt.Println("City current population:", currentPopulation)
	fmt.Println("City total population:", totalPopulation)

	buffer1 := readFixedList(streamReader, 10)
	cityName := readVarString(streamReader, "CityName")
	fmt.Println("Buffer1:", buffer1, ", cityName:", cityName)

	flagBeforeRewards := unsafeReadUint8(streamReader)
	if flagBeforeRewards != 0 {
		log.Fatal("flagBeforeRewards isn't 0")
	}
	cityRewardsSize := unsafeReadUint16(streamReader)
	cityRewards := make([]int, cityRewardsSize)
	for i := 0; i < int(cityRewardsSize); i++ {
		cityReward := unsafeReadUint16(streamReader)
		cityRewards[i] = int(cityReward)
	}
	fmt.Println("CityRewards:", cityRewards)

	rebellionFlag := unsafeReadUint16(streamReader)
	fmt.Println("Rebellion flag:", rebellionFlag)
	if rebellionFlag != 0 {
		buffer := readFixedList(streamReader, 2)
		fmt.Println("bufferRebel:", buffer)
	}

	unitFlag := unsafeReadUint8(streamReader)
	fmt.Println("Unit flag:", unitFlag)
	var unitDataPtr *UnitData
	if unitFlag == 1 {
		fmt.Println("City has unit")
		unitData := UnitData{}
		if err := binary.Read(streamReader, binary.LittleEndian, &unitData); err != nil {
			log.Fatal("Failed to load buffer: ", err)
		}
		fmt.Println("Unit data:", unitData)
		unitDataPtr = &unitData

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
		fmt.Println(fmt.Sprintf("flag1: %v, flag2: %v, bufferUnit: %v", flag1, flag2, bufferUnit))
	}

	playerVisibilityListSize := unsafeReadUint8(streamReader)
	playerVisibilityList := readFixedList(streamReader, int(playerVisibilityListSize))
	fmt.Println("PlayerVisibilityList:", playerVisibilityList)

	unknown := readFixedList(streamReader, 6)
	fmt.Println("Unknown:", unknown)

	return TileData{
		Terrain:  int(tileDataHeader.Terrain),
		Climate:  int(tileDataHeader.Climate),
		Owner:    int(tileDataHeader.Owner),
		Capital:  int(tileDataHeader.Capital),
		HasCity:  true,
		CityName: cityName,
		Unit:     unitDataPtr,
	}
}

func readOtherTile(streamReader *io.SectionReader, tileDataHeader TileDataHeader, resourceType int, improvementType int) TileData {
	// Has improvement
	if improvementType != -1 {
		// Read improvement data
		improvementData := ImprovementData{}
		if err := binary.Read(streamReader, binary.LittleEndian, &improvementData); err != nil {
			log.Fatal("Failed to load buffer: ", err)
		}
		fmt.Println("Remaining improvement data:", improvementData)
	}

	// Read unit data
	hasUnitFlag := unsafeReadUint8(streamReader)
	var unitDataPtr *UnitData
	if hasUnitFlag == 1 {
		fmt.Println("Tile has unit")
		unitData := UnitData{}
		if err := binary.Read(streamReader, binary.LittleEndian, &unitData); err != nil {
			log.Fatal("Failed to load buffer: ", err)
		}
		fmt.Println("Unit data:", unitData)
		unitDataPtr = &unitData

		hasOtherUnitFlag := unsafeReadUint8(streamReader)
		if hasOtherUnitFlag == 1 {
			// If unit embarks or disembarks, a new unit is created in the backend, but it's still the same unit in the game
			fmt.Println("Tile has previous unit data before transition")
			previousUnitData := UnitData{}
			if err := binary.Read(streamReader, binary.LittleEndian, &previousUnitData); err != nil {
				log.Fatal("Failed to load buffer: ", err)
			}
			fmt.Println("Previous unit data:", previousUnitData)

			flag1 := unsafeReadUint8(streamReader)
			fmt.Println("Flag1:", flag1)
			bufferUnitData2 := readFixedList(streamReader, 7)
			fmt.Println("bufferUnitData2:", bufferUnitData2)

			bufferUnitData3 := readFixedList(streamReader, 7)
			if bufferUnitData2[0] == 1 {
				bufferUnitData3Remainder := readFixedList(streamReader, 4)
				bufferUnitData3 = append(bufferUnitData3, bufferUnitData3Remainder...)
			}
			fmt.Println("bufferUnitData3:", bufferUnitData3)
		} else {
			bufferUnitFlag := unsafeReadUint8(streamReader)
			bufferSize := 6
			if bufferUnitFlag == 1 {
				bufferSize = 8
			}

			bufferUnit := readFixedList(streamReader, bufferSize)
			fmt.Println("bufferUnit:", bufferUnit)
		}
	}

	playerVisibilityListSize := unsafeReadUint8(streamReader)
	playerVisibilityList := readFixedList(streamReader, int(playerVisibilityListSize))
	fmt.Println("PlayerVisibilityList:", playerVisibilityList)

	unknown := readFixedList(streamReader, 6)
	fmt.Println("Unknown:", unknown)

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
		Unit:     unitDataPtr,
	}
}

func readTileData(streamReader *io.SectionReader, tileData [][]TileData, mapWidth int, mapHeight int) {
	allUnitData := make([]UnitData, 0)

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

			if tileData[i][j].Unit != nil {
				allUnitData = append(allUnitData, *tileData[i][j].Unit)
			}
		}
	}

	fmt.Println("Total number of units:", len(allUnitData))
	for i := 0; i < len(allUnitData); i++ {
		fmt.Println("Unit", i, ":", allUnitData[i])
	}
}

func readMapHeader(streamReader *io.SectionReader) MapHeaderOutput {
	mapHeader := MapHeaderInput{}
	if err := binary.Read(streamReader, binary.LittleEndian, &mapHeader); err != nil {
		log.Fatal("Failed to load MapHeaderInput: ", err)
	}
	fmt.Println("Map header:", mapHeader)

	mapName := readVarString(streamReader, "MapName")
	fmt.Println("Map name:", mapName)

	// map dimenions is a square: squareSize x squareSize
	squareSize := int(unsafeReadUint32(streamReader))
	fmt.Println("Map square size:", squareSize)

	disabledTribesSize := unsafeReadUint16(streamReader)
	disabledTribesArr := make([]int, disabledTribesSize)
	if disabledTribesSize > 0 {
		for i := 0; i < int(disabledTribesSize); i++ {
			disabledTribesArr[i] = int(unsafeReadUint16(streamReader))
		}
	}
	fmt.Println("Disabled tribes:", disabledTribesArr, ", size:", disabledTribesSize)

	unlockedTribesSize := unsafeReadUint32(streamReader)
	unlockedTribesArr := make([]int, unlockedTribesSize-1)
	for i := 0; i < int(unlockedTribesSize)-1; i++ {
		unlockedTribesArr[i] = int(unsafeReadUint16(streamReader))
	}
	fmt.Println("Unlocked tribes:", unlockedTribesArr, ", size:", unlockedTribesSize)

	gameDifficulty := unsafeReadUint16(streamReader)
	fmt.Println("Game difficulty:", gameDifficulty)

	numOpponents := unsafeReadUint32(streamReader)
	fmt.Println("Num opponents:", numOpponents)

	unknownArr := readFixedList(streamReader, 5+int(unlockedTribesSize))
	fmt.Println("Unknown:", unknownArr, ", size:", len(unknownArr))

	selectedTribeSkinSize := unsafeReadUint32(streamReader)
	fmt.Println("selectedTribeSkinSize:", selectedTribeSkinSize)

	selectedTribeSkins := make(map[int]int)
	for i := 0; i < int(selectedTribeSkinSize); i++ {
		tribe := unsafeReadUint16(streamReader)
		skin := unsafeReadUint16(streamReader)
		fmt.Println(fmt.Sprintf("Tribe: %v, skin: %v", tribe, skin))

		selectedTribeSkins[int(tribe)] = int(skin)
	}

	mapWidth := unsafeReadUint16(streamReader)
	mapHeight := unsafeReadUint16(streamReader)
	if mapWidth == 0 && mapHeight == 0 {
		mapWidth = unsafeReadUint16(streamReader)
		mapHeight = unsafeReadUint16(streamReader)
	}

	fmt.Println("Map width:", mapWidth, ", height:", mapHeight)

	return MapHeaderOutput{
		MapName:           mapName,
		MapSquareSize:     squareSize,
		DisabledTribesArr: disabledTribesArr,
		UnlockedTribesArr: unlockedTribesArr,
		GameDifficulty:    int(gameDifficulty),
		NumOpponents:      int(numOpponents),
		UnknownArr:        unknownArr,
		SelectedTribes:    selectedTribeSkins,
		MapWidth:          int(mapWidth),
		MapHeight:         int(mapHeight),
	}
}

func readPlayerData(streamReader *io.SectionReader) PlayerData {
	playerId := unsafeReadUint8(streamReader)
	playerName := readVarString(streamReader, "playerName")
	playerAccountId := readVarString(streamReader, "playerAccountId")
	autoPlay := unsafeReadUint8(streamReader)
	startTileCoordinates1 := unsafeReadInt32(streamReader)
	startTileCoordinates2 := unsafeReadInt32(streamReader)
	tribe := unsafeReadUint16(streamReader)
	unknownByte1 := unsafeReadUint8(streamReader)
	unknownInt1 := unsafeReadUint32(streamReader)

	unknownArrLen1 := unsafeReadUint16(streamReader)
	unknownArr1 := make([]int, 0)
	for i := 0; i < int(unknownArrLen1); i++ {
		value1 := unsafeReadUint8(streamReader)
		value2 := readFixedList(streamReader, 4)
		unknownArr1 = append(unknownArr1, int(value1), int(value2[0]), int(value2[1]), int(value2[2]), int(value2[3]))
	}

	currency := unsafeReadUint32(streamReader)
	score := unsafeReadUint32(streamReader)
	unknownInt2 := unsafeReadUint32(streamReader)
	numCities := unsafeReadUint16(streamReader)

	techArrayLen := unsafeReadUint16(streamReader)
	techArray := make([]int, techArrayLen)
	for i := 0; i < int(techArrayLen); i++ {
		techType := unsafeReadUint16(streamReader)
		techArray[i] = int(techType)
	}

	encounteredPlayersLen := unsafeReadUint16(streamReader)
	encounteredPlayers := make([]int, 0)
	for i := 0; i < int(encounteredPlayersLen); i++ {
		playerId := unsafeReadUint8(streamReader)
		encounteredPlayers = append(encounteredPlayers, int(playerId))
	}

	numTasks := unsafeReadInt16(streamReader)
	taskArr := make([]PlayerTaskData, int(numTasks))
	for i := 0; i < int(numTasks); i++ {
		taskType := unsafeReadInt16(streamReader)

		var buffer []byte
		if taskType == 1 || taskType == 5 { // Task type 1 is Pacifist, type 5 is Killer
			buffer = readFixedList(streamReader, 6) // Extra buffer contains a uint32
		} else if taskType >= 1 && taskType <= 8 {
			buffer = readFixedList(streamReader, 2)
		} else {
			log.Fatal("Invalid task type:", taskType)
		}
		taskArr[i] = PlayerTaskData{
			Type:   int(taskType),
			Buffer: buffer,
		}
	}

	totalKills := unsafeReadInt32(streamReader)
	totalLosses := unsafeReadInt32(streamReader)
	totalTribesDestroyed := unsafeReadInt32(streamReader)
	unknownBuffer1 := readFixedList(streamReader, 5)

	playerUniqueImprovementsSize := unsafeReadUint16(streamReader)
	playerUniqueImprovements := make([]int, int(playerUniqueImprovementsSize))
	for i := 0; i < int(playerUniqueImprovementsSize); i++ {
		improvement := unsafeReadUint16(streamReader)
		playerUniqueImprovements[i] = int(improvement)
	}

	diplomacyArrLen := unsafeReadUint16(streamReader)
	diplomacyArr := make([]DiplomacyData, int(diplomacyArrLen))
	for i := 0; i < len(diplomacyArr); i++ {
		diplomacyData := DiplomacyData{}
		if err := binary.Read(streamReader, binary.LittleEndian, &diplomacyData); err != nil {
			log.Fatal("Failed to load diplomacyData: ", err)
		}
		diplomacyArr[i] = diplomacyData
	}

	diplomacyMessagesSize := unsafeReadUint16(streamReader)
	diplomacyMessagesArr := make([]DiplomacyMessage, int(diplomacyMessagesSize))
	for i := 0; i < int(diplomacyMessagesSize); i++ {
		messageType := unsafeReadUint8(streamReader)
		sender := unsafeReadUint8(streamReader)

		diplomacyMessagesArr[i] = DiplomacyMessage{
			MessageType: int(messageType),
			Sender:      int(sender),
		}
	}

	destroyedByTribe := unsafeReadUint8(streamReader)
	destroyedTurn := unsafeReadUint32(streamReader)
	unknownBuffer2 := readFixedList(streamReader, 14)

	return PlayerData{
		Id:                   int(playerId),
		Name:                 playerName,
		AccountId:            playerAccountId,
		AutoPlay:             int(autoPlay) != 0,
		StartTileCoordinates: [2]int{int(startTileCoordinates1), int(startTileCoordinates2)},
		Tribe:                int(tribe),
		UnknownByte1:         int(unknownByte1),
		UnknownInt1:          int(unknownInt1),
		UnknownArr1:          unknownArr1,
		Currency:             int(currency),
		Score:                int(score),
		UnknownInt2:          int(unknownInt2),
		NumCities:            int(numCities),
		AvailableTech:        techArray,
		EncounteredPlayers:   encounteredPlayers,
		Tasks:                taskArr,
		TotalUnitsKilled:     int(totalKills),
		TotalUnitsLost:       int(totalLosses),
		TotalTribesDestroyed: int(totalTribesDestroyed),
		UnknownBuffer1:       unknownBuffer1,
		UniqueImprovements:   playerUniqueImprovements,
		DiplomacyArr:         diplomacyArr,
		DiplomacyMessages:    diplomacyMessagesArr,
		DestroyedByTribe:     int(destroyedByTribe),
		DestroyedTurn:        int(destroyedTurn),
		UnknownBuffer2:       unknownBuffer2,
	}
}

func readAllPlayerData(streamReader *io.SectionReader) []PlayerData {
	numPlayers := unsafeReadUint16(streamReader)
	fmt.Println("Num players:", numPlayers)
	allPlayerData := make([]PlayerData, int(numPlayers))

	for i := 0; i < int(numPlayers); i++ {
		playerData := readPlayerData(streamReader)
		allPlayerData[i] = playerData
		fmt.Printf("%+v\n", playerData)
	}
	return allPlayerData
}

func buildOwnerTribeMap(allPlayerData []PlayerData) map[int]int {
	ownerTribeMap := make(map[int]int)

	for i := 0; i < len(allPlayerData); i++ {
		playerData := allPlayerData[i]
		mappedTribe, ok := ownerTribeMap[playerData.Id]
		if ok {
			log.Fatal(fmt.Sprintf("Owner to tribe map has duplicate player id %v already mapped to %v", playerData.Id, mappedTribe))
		}
		ownerTribeMap[playerData.Id] = playerData.Tribe
	}

	return ownerTribeMap
}

func ReadPolytopiaSaveFile(inputFilename string) (*PolytopiaSaveOutput, error) {
	decompressedReader, decompressedLength := buildReaderForDecompressedFile(inputFilename)
	streamReader := io.NewSectionReader(decompressedReader, int64(0), int64(decompressedLength))

	// Read initial map state
	initialMapHeaderOutput := readMapHeader(streamReader)
	fmt.Println("initialMapHeaderOutput:", initialMapHeaderOutput)

	tileData := make([][]TileData, initialMapHeaderOutput.MapHeight)
	for i := 0; i < initialMapHeaderOutput.MapHeight; i++ {
		tileData[i] = make([]TileData, initialMapHeaderOutput.MapWidth)
	}

	readTileData(streamReader, tileData, initialMapHeaderOutput.MapWidth, initialMapHeaderOutput.MapHeight)

	ownerTribeMap := buildOwnerTribeMap(readAllPlayerData(streamReader))
	fmt.Println("Owner to tribe map:", ownerTribeMap)

	partBetweenInitialAndCurrentMap := readFixedList(streamReader, 3)
	fmt.Println("partBetweenInitialAndCurrentMap:", partBetweenInitialAndCurrentMap)

	// Read current map state
	currentMapHeaderOutput := readMapHeader(streamReader)
	fmt.Println("currentMapHeaderOutput:", currentMapHeaderOutput)

	readTileData(streamReader, tileData, currentMapHeaderOutput.MapWidth, currentMapHeaderOutput.MapHeight)

	ownerTribeMap = buildOwnerTribeMap(readAllPlayerData(streamReader))
	fmt.Println("Owner to tribe map:", ownerTribeMap)

	output := &PolytopiaSaveOutput{
		MapHeight:     initialMapHeaderOutput.MapHeight,
		MapWidth:      initialMapHeaderOutput.MapWidth,
		OwnerTribeMap: ownerTribeMap,
		TileData:      tileData,
	}
	return output, nil
}
