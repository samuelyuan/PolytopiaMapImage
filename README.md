# PolytopiaMapImage

## Introduction

The Battle of Polytopia only allows one save file to be saved. If a new game is started, the old save files are discarded. There is no way to view a snapshot of the game at any time or any replay system that allows you to view past games.

This program extracts the map from the save file and converts it into an image or replay file that you can preserve and share.

## Command-Line Usage

The input filename must be a .state file from the save game directory. Make sure to copy the .state file in a different folder, such as this project, because the .state file will be deleted after you win or lose every game. Once the game ends and the .state file is lost, it can't be recovered.

The output filename is an image or gif that you want to save to.

The mode is either "image" or "replay". The image mode will generate a screenshot of the map at the last saved turn and the replay mode will generate an entire replay of the game from the beginning to the current turn.

```
./PolytopiaMapImage.exe -input=[input filename] -output=[output filename (default is output.png)] -mode=[drawing mode (default is image)]
```

### Draw Image

```
./PolytopiaMapImage.exe -input=00000000-0000-0000-0000-000000000000.state -output=map.png -mode=image]
```

### Draw Replay

```
./PolytopiaMapImage.exe -input=00000000-0000-0000-0000-000000000000.state -output=replay.gif -mode=replay
```

## Examples

Map Image
<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/PolytopiaMapImage/master/examples/map.png" alt="map" width="300" height="300" />
</div>

Replay
<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/PolytopiaMapImage/master/examples/replay.gif" alt="replay" width="300" height="300" />
<img src="https://raw.githubusercontent.com/samuelyuan/PolytopiaMapImage/master/examples/pangea.gif" alt="pangea" width="300" height="300" />
</div>

## File format

The .state file is compressed using LZ4. The file consists of the initial map state, current map state, and a list of all actions taken in game.

### Map Header

The header contains all of the game settings.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | Version1 |
| uint32 | 4 bytes | Version2 |
| uint16 | 2 bytes | TotalActions |
| uint32 | 4 bytes | CurrentTurn |
| uint8 | 1 byte | CurrentPlayerIndex |
| uint32 | 4 bytes | MaxUnitId |
| uint8 | 1 byte | CurrentGameState |
| int32 | 4 bytes | Seed |
| uint32 | 4 bytes | TurnLimit |
| uint32 | 4 bytes | ScoreLimit |
| uint8 | 1 byte | WinByCapital |
| byte[6] | 6 bytes | UnknownSettings |
| uint8 | 1 byte | GameModeBase |
| uint8 | 1 byte | GameModeRules |
| varstring | var bytes | Map name |
| uint32 | 4 bytes | Map square size |
| uint16 | 2 bytes | disabledTribesSize |
| uint16 array | disabledTribesSize*2 bytes | Disabled tribes |
| uint16 | 2 bytes | unlockedTribesSize |
| uint16 array | unlockedTribesSize*2 bytes | Unlocked tribes |
| uint16 | 2 bytes | Game difficulty |
| uint32 | 4 bytes | Number of opponents |
| uint16 | 2 bytes | Game type |
| uint8 | 1 byte | Map preset |
| int32 | 4 bytes | Turn time limit in minutes |
| float32 | 4 bytes | unknownFloat1 |
| float32 | 4 bytes | unknownFloat2 |
| float32 | 4 bytes | baseTimeSeconds |
| byte[4] | 4 bytes | timeSettings |
| uint32 | 4 bytes | selectedTribeSkinSize |
| uint16 array | selectedTribeSkinSize*2 bytes | Tribe to skin map |
| uint16 | 2 bytes | Map width |
| uint16 | 2 bytes | Map height |

### Map Tiles

The first two unsigned shorts contain the map width and height. The following data is a 2D array of tiles.

The tile data contains resource data, improvement data, unit data if they exist for that tile.

#### Tile Data

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32[2] | 8 bytes | WorldCoordinates |
| uint16 | 2 bytes | Terrain |
| uint16 | 2 bytes | Climate |
| int16 | 2 bytes | Altitude |
| uint8 | 1 byte | Owner |
| uint8 | 1 byte | Capital |
| int32[2] | 8 bytes | CapitalCoordinates |
| bool | 1 byte | ResourceExists |
| uint16 | 2 bytes | ResourceType |
| bool | 1 byte | ImprovementExists |
| uint16 | 2 bytes | ImprovementType |
| ImprovementData | sizeof(ImprovementData) | ImprovementData ( if ImprovementExists is true) |
| bool | 1 byte | HasUnitFlag |
| UnitData | sizeof(UnitData) | Unit (if HasUnitFlag is true) |
| bool | 1 byte | HasPassengerUnitFlag (if HasUnitFlag is true) |
| UnitData | sizeof(UnitData) | PassengerUnit (if HasPassengerUnitFlag is true) |
| bool | 1 byte | PassengerUnitHasUnitFlag (if HasPassengerUnitFlag is true, should always be zero because passenger unit can't carry another unit) |
| uint16 | 2 bytes | PassengerUnitEffectDataLength (if HasPassengerUnitFlag is true) |
| unit16 array | PassengerUnitEffectDataLength*2 bytes | PassengerUnitEffectData (if HasPassengerUnitFlag is true) |
| byte[5] | 5 bytes | PassengerUnitDirectionData (if HasPassengerUnitFlag is true) |
| uint16 | 2 bytes | UnitEffectDataLength (if HasUnitFlag is true, flags: 0 - ice, 1 - poison, 2 - boost, 3 - invisible) |
| unit16 array | UnitEffectDataLength*2 bytes | UnitEffectData (if HasUnitFlag is true) |
| byte[5] | 5 bytes | UnitDirectionData (if HasUnitFlag is true, contains direction flag (0 - southwest, 1 - west, 2 - northwest, 3 - north, 4 - northeast, 5 - east, 6 - southwest, 7 - south)) |
| uint8 | 1 byte | PlayerVisibilityLength |
| uint8 array | PlayerVisibilityLength*1 bytes | PlayerVisibility |
| bool | 1 byte | HasRoad |
| bool | 1 byte | HasWaterRoute |
| uint16 | 2 bytes | TileSkin |
| byte[2] | 2 bytes | Unknown |

#### Improvement Data

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint16 | 2 bytes | Level |
| uint16 | 2 bytes | FoundedTurn |
| int16 | 2 bytes | CurrentPopulation |
| uint16 | 2 bytes | TotalPopulation |
| int16 | 2 bytes | Production |
| int16 | 2 bytes | BaseScore |
| int16 | 2 bytes | BorderSize |
| int16 | 2 bytes | UpgradeCount |
| uint8 | 1 byte | ConnectedPlayerCapital |
| uint8 | 1 byte | HasCityName |
| varstring | var bytes | CityName (only if HasCityName is true) |
| uint8 | 1 byte | FoundedTribe |
| uint16 | 2 bytes | CityRewardsLength |
| uint16 array | CityRewardsLength*2 | CityRewards |
| uint16 | 2 bytes | RebellionFlag |
| byte[2] | 2 bytes | RebellionBuffer |

#### Unit Data

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint32 | 4 bytes | UnitId |
| uint8 | 1 byte | Owner |
| uint16 | 2 bytes | UnitType |
| uint32 | 4 bytes | FollowerUnitId (only initialized for cymanti centipedes and segments) |
| uint32 | 4 bytes | LeaderUnitId (only initialized for cymanti centipedes and segments) |
| int32[2] | 8 bytes | CurrentCoordinates |
| int32[2] | 8 bytes | HomeCoordinates |
| uint16 | 2 bytes | Health (should be divided by 10 to get value ingame) |
| uint16 | 2 bytes | PromotionLevel |
| uint16 | 2 bytes | Experience |
| bool | 1 byte | Moved |
| bool | 1 byte | Attacked |
| bool | 1 byte | Flipped |
| uint16 | 2 bytes | CreatedTurn |

### Player Data

A list of all players in the game.

The size of the player list is a uint16 followed by an array of players.

| Type | Size | Description |
| ---- | ---- | ----------- |
| uint8 | 1 byte | PlayerId |
| varstring | var bytes | Name |
| varstring | var bytes | AccountId |
| bool | 1 byte | AutoPlay |
| int32[2] | 8 bytes | StartTileCoordinates |
| uint16 | 2 bytes | Tribe |
| uint8 | 1 byte | UnknownByte1 |
| uint32 | 4 bytes | DifficultyHandicap |
| uint16 | 2 bytes | AggressionsByPlayers length |
| PlayerAggression array | AggressionsByPlayersLength*5 | AggressionsByPlayers |
| uint32 | 4 bytes | Currency |
| uint32 | 4 bytes | Score |
| uint32 | 4 bytes | UnknownInt2 |
| uint16 | 2 bytes | NumCities |
| uint16 | 2 bytes | AvailableTechLength |
| uint16 array | AvailableTechLength*2 | AvailableTech |
| uint16 | 2 bytes | EncounteredPlayersLength |
| uint8 array | EncounteredPlayersLength*1 | EncounteredPlayers |
| uint16 | 2 bytes | PlayerTaskDataLength |
| PlayerTaskData array | variable length depending on task and buffer | PlayerTaskData |
| int32 | 4 bytes | TotalUnitsKilled |
| int32 | 4 bytes | TotalUnitsLost |
| int32 | 4 bytes | TotalTribesDestroyed |
| uint8[4] | 4 bytes | OverrideColor |
| uint8 | 1 byte | OverrideTribe |
| uint16 | 2 bytes | UniqueImprovementsLength |
| uint16 array | UniqueImprovementsLength*2 | UniqueImprovements |
| uint16 | 2 bytes | DiplomacyArrLength |
| DiplomacyData array | DiplomacyArrLength*23 | DiplomacyArr |
| uint16 | 2 bytes | DiplomacyMessagesLength |
| DiplomacyMessage array | DiplomacyMessagesLength*2 | DiplomacyMessages |
| uint8 | 1 byte | DestroyedByTribe |
| uint32 | 4 bytes | DestroyedTurn |
| uint8[4] | 4 bytes | UnknownBuffer2 |
| int32 | 4 bytes | EndScore |
| uint16 | 2 bytes | PlayerSkin |
| uint8[4] | 4 bytes | UnknownBuffer3 |

### Actions List

The first two bytes is an unsigned short that describes how many actions there are saved. The following data is a list of actions.

Every action begins with an unsigned short that describes the type of action followed by a fixed number of bytes depending on the action.
