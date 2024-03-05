# PolytopiaMapImage

## Introduction

The Battle of Polytopia only allows one save file to be saved. If a new game is started, the old save files are discarded. There is no replay system that allows you to view past games. This program extracts the map from the save file and converts it into an image.

## Command-Line Usage

The input filename must be a .state file from the save game directory.

The output filename is an image that you want to save to.

```
./PolytopiaMapImage.exe -input=[input filename] -output=[output filename (default is output.png)]
```

## Examples

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/PolytopiaMapImage/master/screenshots/map.png" alt="map" width="200" height="200" />
</div>

## File format

The .state file is compressed using LZ4.