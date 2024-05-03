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
	Version1           uint32
	Version2           uint32
	TotalActions       uint16
	CurrentTurn        uint32
	CurrentPlayerIndex uint8
	MaxUnitId          uint32
	UnknownByte1       uint8
	Seed               uint32
	TurnLimit          uint32
	Unknown1           [11]byte
	GameMode1          uint8
	GameMode2          uint8
}

type MapHeaderOutput struct {
	MapHeaderInput    MapHeaderInput
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
	WorldCoordinates   [2]int
	Terrain            int
	Climate            int
	Altitude           int
	Owner              int
	Capital            int
	CapitalCoordinates [2]int
	ResourceExists     bool
	ResourceType       int
	ImprovementExists  bool
	ImprovementType    int
	ImprovementData    *ImprovementData
	Unit               *UnitData
	BufferUnitData     []byte
	PlayerVisibility   []uint8
	HasRoad            bool
	HasWaterRoute      bool
	Unknown            []byte
}

type ImprovementData struct {
	Level                  int
	FoundedTurn            int
	CurrentPopulation      int
	TotalPopulation        int
	Production             int
	BaseScore              int
	BorderSize             int // For cities, 1 is default, 2 is expanded border
	UpgradeCount           int // For cities, seems to be -1 * (level - 1). Level 1 is starting point and no upgrades.
	ConnectedPlayerCapital int
	HasCityName            int
	CityName               string
	FoundedTribe           int
	CityRewards            []int
	RebellionFlag          int
	RebellionBuffer        []byte
}

type ActionBuild struct {
	PlayerId        uint8
	ImprovementType uint16
	Coordinates     [2]uint32
}

type ActionAttack struct {
	PlayerId uint8
	UnitId   uint32
	Origin   [2]uint32
	Target   [2]uint32
}

type ActionRecover struct {
	PlayerId    uint8
	Coordinates [2]uint32
}

type ActionTrain struct {
	PlayerId uint8
	UnitType uint16
	Position [2]uint32
}

type ActionMove struct {
	PlayerId    uint8
	OldPosition [2]uint32
	NewPosition [2]uint32
	UnitId      uint32
}

type ActionCaptureCity struct {
	PlayerId    uint8
	UnitId      uint32
	Coordinates [2]uint32
}

type ActionResearch struct {
	PlayerId uint8
	TechType uint16
}

type ActionDestroyImprovement struct {
	PlayerId    uint8
	Coordinates [2]uint32
}

type ActionCityReward struct {
	PlayerId    uint8
	Coordinates [2]uint32
	Reward      uint16
}

type ActionPromote struct {
	PlayerId    uint8
	Coordinates [2]uint32
}

type ActionExamineRuins struct {
	PlayerId    uint8
	Coordinates [2]uint32
}

type ActionEndTurn struct {
	PlayerId uint8
}

type ActionUpgrade struct {
	PlayerId    uint8
	UnitType    uint16
	Coordinates [2]uint32
}

type ActionCityLevelUp struct {
	PlayerId    uint8
	Coordinates [2]uint32
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
	ColorOverride        []byte
	UnknownByte2         byte
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
	MapHeight       int
	MapWidth        int
	PlayerData      []PlayerData
	OwnerTribeMap   map[int]int
	InitialTileData [][]TileData
	TileData        [][]TileData
	MaxTurn         int
	TurnCaptureMap  map[int][]ActionCaptureCity
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

func unsafeReadInt8(reader *io.SectionReader) int8 {
	signedIntValue := int8(0)
	if err := binary.Read(reader, binary.LittleEndian, &signedIntValue); err != nil {
		log.Fatal("Failed to load int8: ", err)
	}
	return signedIntValue
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

func readImprovementData(streamReader *io.SectionReader) ImprovementData {
	level := unsafeReadUint16(streamReader)
	foundedTurn := unsafeReadUint16(streamReader)
	currentPopulation := unsafeReadInt16(streamReader)
	totalPopulation := unsafeReadUint16(streamReader)
	production := unsafeReadUint16(streamReader)
	baseScore := unsafeReadUint16(streamReader)
	borderSize := unsafeReadUint16(streamReader)
	upgradeCount := unsafeReadInt16(streamReader)
	connectedPlayerCapital := unsafeReadUint8(streamReader)
	hasCityName := unsafeReadUint8(streamReader)
	cityName := ""
	if hasCityName == 1 {
		cityName = readVarString(streamReader, "CityName")
	}

	foundedTribe := unsafeReadUint8(streamReader)

	cityRewardsSize := unsafeReadUint16(streamReader)
	cityRewards := make([]int, cityRewardsSize)
	for i := 0; i < int(cityRewardsSize); i++ {
		cityReward := unsafeReadUint16(streamReader)
		cityRewards[i] = int(cityReward)
	}

	rebellionFlag := unsafeReadUint16(streamReader)
	var rebellionBuffer []byte
	if rebellionFlag != 0 {
		rebellionBuffer = readFixedList(streamReader, 2)
	}

	return ImprovementData{
		Level:                  int(level),
		FoundedTurn:            int(foundedTurn),
		CurrentPopulation:      int(currentPopulation),
		TotalPopulation:        int(totalPopulation),
		Production:             int(production),
		BaseScore:              int(baseScore),
		BorderSize:             int(borderSize),
		UpgradeCount:           int(upgradeCount),
		ConnectedPlayerCapital: int(connectedPlayerCapital),
		HasCityName:            int(hasCityName),
		CityName:               cityName,
		FoundedTribe:           int(foundedTribe),
		CityRewards:            cityRewards,
		RebellionFlag:          int(rebellionFlag),
		RebellionBuffer:        rebellionBuffer,
	}
}

func readTileData(streamReader *io.SectionReader, tileData [][]TileData, mapWidth int, mapHeight int) {
	allUnitData := make([]UnitData, 0)

	for i := 0; i < int(mapHeight); i++ {
		for j := 0; j < int(mapWidth); j++ {
			tileDataHeader := TileDataHeader{}
			if err := binary.Read(streamReader, binary.LittleEndian, &tileDataHeader); err != nil {
				log.Fatal("Failed to load tileDataHeader: ", err)
			}

			// Sanity check
			if int(tileDataHeader.WorldCoordinates[0]) != j || int(tileDataHeader.WorldCoordinates[1]) != i {
				log.Fatal(fmt.Sprintf("File reached unexpected location. Iteration (%v, %v) isn't equal to world coordinates (%v, %v)",
					i, j, tileDataHeader.WorldCoordinates[0], tileDataHeader.WorldCoordinates[1]))
			}

			resourceExistsFlag := unsafeReadUint8(streamReader)
			resourceType := -1
			if resourceExistsFlag == 1 {
				resourceType = int(unsafeReadUint16(streamReader))
			}

			improvementExistsFlag := unsafeReadUint8(streamReader)
			improvementType := -1
			if improvementExistsFlag == 1 {
				improvementType = int(unsafeReadUint16(streamReader))
			}

			var improvementDataPtr *ImprovementData
			if improvementType != -1 {
				improvementData := readImprovementData(streamReader)
				fmt.Printf("Improvement data: %+v\n", improvementData)
				improvementDataPtr = &improvementData
			}

			// Read unit data
			hasUnitFlag := unsafeReadUint8(streamReader)
			var unitDataPtr *UnitData
			bufferUnitData := make([]byte, 0)
			if hasUnitFlag == 1 {
				unitData := UnitData{}
				if err := binary.Read(streamReader, binary.LittleEndian, &unitData); err != nil {
					log.Fatal("Failed to load buffer: ", err)
				}
				fmt.Printf("Unit data: %+v\n", unitData)
				unitDataPtr = &unitData

				hasOtherUnitFlag := unsafeReadUint8(streamReader)
				if hasOtherUnitFlag == 1 {
					// If unit embarks or disembarks, a new unit is created in the backend, but it's still the same unit in the game
					fmt.Println("Tile has previous unit data before transition")
					previousUnitData := UnitData{}
					if err := binary.Read(streamReader, binary.LittleEndian, &previousUnitData); err != nil {
						log.Fatal("Failed to load buffer: ", err)
					}
					fmt.Printf("Previous unit data: %+v\n", previousUnitData)

					_ = unsafeReadUint8(streamReader)
					bufferUnitData2 := readFixedList(streamReader, 7)
					bufferUnitData3 := readFixedList(streamReader, 7)
					if bufferUnitData2[0] == 1 {
						bufferUnitData3Remainder := readFixedList(streamReader, 4)
						bufferUnitData3 = append(bufferUnitData3, bufferUnitData3Remainder...)
						fmt.Println("Longer buffer3")
					}
					bufferUnitData = append(bufferUnitData, bufferUnitData2...)
					bufferUnitData = append(bufferUnitData, bufferUnitData3...)
					fmt.Println(fmt.Sprintf("bufferUnit2: %v, bufferUnit3: %v", bufferUnitData2, bufferUnitData3))
				} else {
					bufferUnitFlag := unsafeReadUint8(streamReader)
					bufferSize := 6
					if bufferUnitFlag == 1 {
						bufferSize = 8
					}

					bufferUnitData = readFixedList(streamReader, bufferSize)
					fmt.Println(fmt.Sprintf("bufferUnitFlag: %v, bufferUnit: %v", bufferUnitFlag, bufferUnitData))
				}
			}

			playerVisibilityListSize := unsafeReadUint8(streamReader)
			playerVisibilityList := readFixedList(streamReader, int(playerVisibilityListSize))
			hasRoad := unsafeReadUint8(streamReader)
			hasWaterRoute := unsafeReadUint8(streamReader)
			unknown := readFixedList(streamReader, 4)

			tileData[i][j] = TileData{
				WorldCoordinates:   [2]int{int(tileDataHeader.WorldCoordinates[0]), int(tileDataHeader.WorldCoordinates[1])},
				Terrain:            int(tileDataHeader.Terrain),
				Climate:            int(tileDataHeader.Climate),
				Altitude:           int(tileDataHeader.Altitude),
				Owner:              int(tileDataHeader.Owner),
				Capital:            int(tileDataHeader.Capital),
				CapitalCoordinates: [2]int{int(tileDataHeader.CapitalCoordinates[0]), int(tileDataHeader.CapitalCoordinates[1])},
				ImprovementData:    improvementDataPtr,
				Unit:               unitDataPtr,
				BufferUnitData:     bufferUnitData,
				PlayerVisibility:   playerVisibilityList,
				HasRoad:            hasRoad != 0,
				HasWaterRoute:      hasWaterRoute != 0,
				Unknown:            unknown,
				ResourceExists:     resourceExistsFlag != 0,
				ResourceType:       resourceType,
				ImprovementExists:  improvementExistsFlag != 0,
				ImprovementType:    improvementType,
			}
			fmt.Printf(fmt.Sprintf("TileData (%v, %v): %+v\n", i, j, tileData[i][j]))

			if tileData[i][j].Unit != nil {
				allUnitData = append(allUnitData, *tileData[i][j].Unit)
			}
		}
	}

	fmt.Println("Total number of units:", len(allUnitData))
	for i := 0; i < len(allUnitData); i++ {
		fmt.Println(fmt.Sprintf("Unit %v: %+v", i, allUnitData[i]))
	}
}

func readMapHeader(streamReader *io.SectionReader) MapHeaderOutput {
	mapHeaderInput := MapHeaderInput{}
	if err := binary.Read(streamReader, binary.LittleEndian, &mapHeaderInput); err != nil {
		log.Fatal("Failed to load MapHeaderInput: ", err)
	}

	mapName := readVarString(streamReader, "MapName")

	// map dimenions is a square: squareSize x squareSize
	squareSize := int(unsafeReadUint32(streamReader))

	disabledTribesSize := unsafeReadUint16(streamReader)
	disabledTribesArr := make([]int, disabledTribesSize)
	for i := 0; i < int(disabledTribesSize); i++ {
		disabledTribesArr[i] = int(unsafeReadUint16(streamReader))
	}

	unlockedTribesSize := unsafeReadUint16(streamReader)
	unlockedTribesArr := make([]int, unlockedTribesSize)
	for i := 0; i < int(unlockedTribesSize); i++ {
		unlockedTribesArr[i] = int(unsafeReadUint16(streamReader))
	}

	gameDifficulty := unsafeReadUint16(streamReader)
	numOpponents := unsafeReadUint32(streamReader)
	unknownArr := readFixedList(streamReader, 5+int(unlockedTribesSize))

	selectedTribeSkinSize := unsafeReadUint32(streamReader)
	selectedTribeSkins := make(map[int]int)
	for i := 0; i < int(selectedTribeSkinSize); i++ {
		tribe := unsafeReadUint16(streamReader)
		skin := unsafeReadUint16(streamReader)
		selectedTribeSkins[int(tribe)] = int(skin)
	}

	mapWidth := unsafeReadUint16(streamReader)
	mapHeight := unsafeReadUint16(streamReader)
	if mapWidth == 0 && mapHeight == 0 {
		mapWidth = unsafeReadUint16(streamReader)
		mapHeight = unsafeReadUint16(streamReader)
	}

	return MapHeaderOutput{
		MapHeaderInput:    mapHeaderInput,
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
	colorOverride := readFixedList(streamReader, 4)
	unknownByte2 := unsafeReadUint8(streamReader)

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
		ColorOverride:        colorOverride,
		UnknownByte2:         unknownByte2,
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

func readAllActions(streamReader *io.SectionReader) map[int][]ActionCaptureCity {
	numActions := unsafeReadUint16(streamReader)
	fmt.Println("Total actions:", numActions)

	turnCaptureMap := make(map[int][]ActionCaptureCity)

	fmt.Println("Reading all player actions:")
	turn := 1
	for i := 0; i < int(numActions); i++ {
		actionType := unsafeReadUint16(streamReader)

		var buffer []byte
		if actionType == 1 {
			action := ActionBuild{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("Build: %+v\n", action)
		} else if actionType == 2 {
			action := ActionAttack{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("Attack: %+v\n", action)
		} else if actionType == 3 {
			action := ActionRecover{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("Recover: %+v\n", action)
		} else if actionType == 4 {
			buffer = readFixedList(streamReader, 9)
		} else if actionType == 5 {
			action := ActionTrain{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("Train: %+v\n", action)
		} else if actionType == 6 {
			action := ActionMove{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("Move: %+v\n", action)
		} else if actionType == 7 {
			action := ActionCaptureCity{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("CaptureCity: %+v\n", action)

			_, ok := turnCaptureMap[turn]
			if !ok {
				turnCaptureMap[turn] = make([]ActionCaptureCity, 0)
			}
			turnCaptureMap[turn] = append(turnCaptureMap[turn], action)
		} else if actionType == 8 {
			action := ActionResearch{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("Research: %+v\n", action)
		} else if actionType == 9 {
			action := ActionDestroyImprovement{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("DestroyImprovement: %+v\n", action)
		} else if actionType == 11 {
			action := ActionCityReward{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("CityReward: %+v\n", action)
		} else if actionType == 13 {
			action := ActionPromote{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("Promote: %+v\n", action)
		} else if actionType == 14 {
			action := ActionExamineRuins{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("ExamineRuins: %+v\n", action)
		} else if actionType == 15 {
			action := ActionEndTurn{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("EndTurn: %+v\n", action)

			if action.PlayerId == 255 {
				turn++
				fmt.Println("Start new turn", turn)
			}
		} else if actionType == 16 {
			action := ActionUpgrade{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("Upgrade: %+v\n", action)
		} else if actionType == 17 {
			buffer = readFixedList(streamReader, 9)
		} else if actionType == 18 {
			buffer = readFixedList(streamReader, 9)
		} else if actionType == 20 {
			buffer = readFixedList(streamReader, 1)
		} else if actionType == 21 {
			action := ActionCityLevelUp{}
			if err := binary.Read(streamReader, binary.LittleEndian, &action); err != nil {
				log.Fatal("Failed to load action: ", err)
			}
			fmt.Printf("CityLevelUp: %+v\n", action)
		} else if actionType == 24 {
			buffer = readFixedList(streamReader, 9)
		} else if actionType == 27 {
			buffer = readFixedList(streamReader, 10)
		} else if actionType == 28 {
			buffer = readFixedList(streamReader, 3)
		} else if actionType == 29 {
			buffer = readFixedList(streamReader, 10)
		} else if actionType == 30 {
			buffer = readFixedList(streamReader, 10)
		} else {
			log.Fatal("Unknown action type:", actionType)
		}

		if len(buffer) > 0 {
			fmt.Println("Index", i, ", action type:", actionType, ", buffer:", buffer)
		}
	}

	return turnCaptureMap
}

func ReadPolytopiaSaveFile(inputFilename string) (*PolytopiaSaveOutput, error) {
	decompressedReader, decompressedLength := buildReaderForDecompressedFile(inputFilename)
	streamReader := io.NewSectionReader(decompressedReader, int64(0), int64(decompressedLength))

	// Read initial map state
	initialMapHeaderOutput := readMapHeader(streamReader)
	fmt.Printf("initialMapHeaderOutput: %+v\n", initialMapHeaderOutput)

	initialTileData := make([][]TileData, initialMapHeaderOutput.MapHeight)
	for i := 0; i < initialMapHeaderOutput.MapHeight; i++ {
		initialTileData[i] = make([]TileData, initialMapHeaderOutput.MapWidth)
	}
	readTileData(streamReader, initialTileData, initialMapHeaderOutput.MapWidth, initialMapHeaderOutput.MapHeight)

	ownerTribeMap := buildOwnerTribeMap(readAllPlayerData(streamReader))
	fmt.Println("Owner to tribe map:", ownerTribeMap)

	partBetweenInitialAndCurrentMap := readFixedList(streamReader, 3)
	fmt.Println("partBetweenInitialAndCurrentMap:", partBetweenInitialAndCurrentMap)

	// Read current map state
	currentMapHeaderOutput := readMapHeader(streamReader)
	fmt.Printf("currentMapHeaderOutput: %+v\n", currentMapHeaderOutput)

	tileData := make([][]TileData, currentMapHeaderOutput.MapHeight)
	for i := 0; i < currentMapHeaderOutput.MapHeight; i++ {
		tileData[i] = make([]TileData, currentMapHeaderOutput.MapWidth)
	}
	readTileData(streamReader, tileData, currentMapHeaderOutput.MapWidth, currentMapHeaderOutput.MapHeight)

	playerData := readAllPlayerData(streamReader)
	ownerTribeMap = buildOwnerTribeMap(playerData)
	fmt.Println("Owner to tribe map:", ownerTribeMap)

	partAfterCurrentMap := readFixedList(streamReader, 2)
	fmt.Println("PartAfterCurrentMap:", partAfterCurrentMap)

	turnCaptureMap := readAllActions(streamReader)

	output := &PolytopiaSaveOutput{
		MapHeight:       currentMapHeaderOutput.MapHeight,
		MapWidth:        currentMapHeaderOutput.MapWidth,
		PlayerData:      playerData,
		OwnerTribeMap:   ownerTribeMap,
		InitialTileData: initialTileData,
		TileData:        tileData,
		MaxTurn:         int(currentMapHeaderOutput.MapHeaderInput.CurrentTurn),
		TurnCaptureMap:  turnCaptureMap,
	}
	return output, nil
}
