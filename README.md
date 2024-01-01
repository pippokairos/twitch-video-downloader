# Twitch Video Downloader

Script that downloads the last VOD from a Twitch channel and uploads it to a private Google Drive. It requires [TwitchDownloader](https://github.com/lay295/TwitchDownloader) installed on your machine.

# Why?
- I want to play with Go
- I want to download the previous night VOD of a streamer to watch it while commuting, since the connection in the train is bad...

## Prerequisites

- Install [Go](https://go.dev)
- Install [TwitchDownloader](https://github.com/lay295/TwitchDownloader)
- On Google Cloud
  - Create a new project
  - Add the private Gmail user address to the test users
  - Add the `Google Drive API` service
  - Create a service account
  - Generate and store a JSON key for that account

## Clone the project

```
git clone https://github.com/pippokairos/twitch-video-downloader.git
```

## Set the environment variables

Just copy the [.env.example](.env.example) file to `.env` and fill the variables.

# Build the executable

```
go build
```

## Run the script

```
./twitch-video-downloader
```

With options:

<table>

<thead>
<tr>
<th>Option</th>
<th>Description</th>
<th>Required</th>
<th>Default</th>
</tr>
</thead>

<tbody>
<tr>
<td>`-name`</td>
<td>The Twitch channel name</td>
<td>True</td>
<td></td>
</tr>

<tr>
<td>`-quality`</td>
<td>Set the video quality</td>
<td>False</td>
<td>720p60</td>
</tr>

<tr>
<td>`-start`</td>
<td>Set the video start time in seconds</td>
<td>False</td>
<td>0</td>
</tr>

<tr>
<td>`-end`</td>
<td>Set the video end time in second</td>
<td>False</td>
<td>[last second]</td>
</tr>
</tbody>

</table>

Example:

```
./twitch-video-downloader -name dada -quality 360p -start 3600 -end 7200
```
*Note: the script will delete all the other files from that channel*
```
TwitchDownloaderCLI 1.53.2 Copyright (c) 2019 lay295 and contributors
[STATUS] - Fetching Video Info [1/5]
[STATUS] - Downloading 100% [2/5]
[STATUS] - Verifying Parts 100% [3/5]
[STATUS] - Combining Parts 100% [4/5]
[STATUS] - Finalizing Video 100% [5/5]
Parent folder found
File uploaded successfully. File ID: 4FNVZGhWFCIxDFthAj_OPUKJ1WN50oYel
File shared successfully with example@gmail.com
Old file deleted successfully: Dada - Un 2023 strano ma pieno di cutzi !minecraft | !zabayo !prime !airbnb !mods.mp4
```

## Contribute...?

This is just a very simple application I wrote to play with Go, there are already more complete solutions out there and I'm not aiming to expand this library's capabilities so there's no guideline for contributing. But hey, if you'd like to make additions, feel free to do so.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
