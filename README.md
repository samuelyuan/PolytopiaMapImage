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

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/PolytopiaMapImage/master/examples/map.png" alt="map" width="300" height="300" />
</div>

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/PolytopiaMapImage/master/examples/replay.gif" alt="replay" width="300" height="300" />
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
| uint8 | 1 byte | UnknownByte1 |
| uint32 | 4 bytes | Seed |
| uint32 | 4 bytes | TurnLimit |
| byte[11] | 11 bytes | Unknown1 |
| uint8 | 1 byte | GameMode1 |
| uint8 | 1 byte | GameMode2 |
| varstring | var bytes | Map name |
| uint32 | 4 bytes | Map square size |
| uint16 | 2 bytes | disabledTribesSize |
| uint16 array | disabledTribesSize*2 bytes | Disabled tribes |
| uint16 | 2 bytes | unlockedTribesSize |
| uint16 array | unlockedTribesSize*2 bytes | Unlocked tribes |
| uint16 | 2 bytes | Game difficulty |
| uint32 | 4 bytes | Number of opponents |
| byte array | 5+unlockedTribesSize bytes | Unknown |
| uint32 | 4 bytes | selectedTribeSkinSize |
| uint16 array | selectedTribeSkinSize*2 bytes | Tribe to skin map |
| uint16 | 2 bytes | Map width |
| uint16 | 2 bytes | Map height |

### Map Tiles

The first two unsigned shorts contain the map width and height. The following data is a 2D array of tiles.

### Player Data

A list of all players in the game.

### Actions List

The first two bytes is an unsigned short that describes how many actions there are saved. The following data is a list of actions.

Every action begins with an unsigned short that describes the type of action followed by a fixed number of bytes depending on the action.
