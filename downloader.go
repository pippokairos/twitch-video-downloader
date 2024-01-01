package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type UsersResponse struct {
	Data []UserData `json:"data"`
}

type UserData struct {
	Id              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageUrl string `json:"profile_image_url"`
	OfflineImageUrl string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	Email           string `json:"email"`
	CreatedAd       string `json:"created_at"`
}

type VideosResponse struct {
	Data []VideosData `json:"data"`
}

type VideosData struct {
	ID            string         `json:"id"`
	StreamID      interface{}    `json:"stream_id"`
	UserID        string         `json:"user_id"`
	UserLogin     string         `json:"user_login"`
	UserName      string         `json:"user_name"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	CreatedAt     string         `json:"created_at"`
	PublishedAt   string         `json:"published_at"`
	URL           string         `json:"url"`
	ThumbnailURL  string         `json:"thumbnail_url"`
	Viewable      string         `json:"viewable"`
	ViewCount     int            `json:"view_count"`
	Language      string         `json:"language"`
	Type          string         `json:"type"`
	Duration      string         `json:"duration"`
	MutedSegments []MutedSegment `json:"muted_segments"`
}

type MutedSegment struct {
	Duration int `json:"duration"`
	Offset   int `json:"offset"`
}

func main() {
	loadEnvVariables()

	name := flag.String("name", "", "The channel name.")
	quality := flag.String("quality", "", "The video quality.")
	start := flag.String("start", "", "The video start in seconds.")
	end := flag.String("end", "", "The video end in seconds.")
	flag.Parse()

	if *name == "" {
		panic("Please, specify a channel name")
	}

	// GET Access Token
	//
	// https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/#client-credentials-flow-example
	authData := url.Values{}
	authData.Set("client_id", os.Getenv("TWITCH_CLIENT_ID"))
	authData.Set("client_secret", os.Getenv("TWITCH_CLIENT_SECRET"))
	authData.Set("grant_type", "client_credentials")

	res, err := http.PostForm("https://id.twitch.tv/oauth2/token", authData)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer res.Body.Close()

	// Read the response body into a slice of bytes
	authBodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading auth response body:", err)
		return
	}

	// Parse the JSON response
	var tokenResponse TokenResponse
	err = json.Unmarshal(authBodyBytes, &tokenResponse)
	if err != nil {
		fmt.Println("Error decoding auth response body to JSON:", err)
		return
	}

	accessToken := tokenResponse.AccessToken

	client := &http.Client{}

	// GET Users
	//
	// https://dev.twitch.tv/docs/api/reference/#get-users
	userUrl := "https://api.twitch.tv/helix/users?login=" + *name
	req, err := http.NewRequest("GET", userUrl, nil)
	if err != nil {
		fmt.Println("Error creating the user GET request:", err)
		return
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Client-Id", os.Getenv("TWITCH_CLIENT_ID"))
	res, err = client.Do(req)
	if err != nil {
		fmt.Println("Error making the user http request:", err)
		return
	}

	// Read the response body into a slice of bytes
	usersBodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading users response body:", err)
		return
	}

	// Parse the JSON response
	var UsersResponse UsersResponse
	err = json.Unmarshal(usersBodyBytes, &UsersResponse)
	if err != nil {
		fmt.Println("Error decoding users response body to JSON:", err)
		return
	}

	// GET Video
	//
	// https://dev.twitch.tv/docs/api/reference/#get-videos
	videosUrl := "https://api.twitch.tv/helix/videos?type=archive&sort=time&first=1&user_id=" + UsersResponse.Data[0].Id
	req, err = http.NewRequest("GET", videosUrl, nil)
	if err != nil {
		fmt.Println("Error creating the videos GET request:", err)
		return
	}

	// TODO: Make func to be DRYer
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Client-Id", os.Getenv("TWITCH_CLIENT_ID"))
	res, err = client.Do(req)
	if err != nil {
		fmt.Println("Error making the videos http request:", err)
		return
	}

	// Read the response body into a slice of bytes
	videosBodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading videos response body:", err)
		return
	}

	// Parse the JSON response
	var VideosResponse VideosResponse
	err = json.Unmarshal(videosBodyBytes, &VideosResponse)
	if err != nil {
		fmt.Println("Error decoding videos response body to JSON:", err)
		return
	}

	// Download the video
	twitchDownloaderCLIPath := os.Getenv("TWITCH_DOWNLOADER_CLI_PATH")
	videoData := VideosResponse.Data[0]
	fileName := videoData.UserName + " - " + videoData.Title + ".mp4"
	filePath := os.Getenv("LOCAL_FILE_PATH") + fileName

	downloaderArgs := []string{"videodownload", "--id", videoData.ID, "-o", filePath}

	if *quality != "" {
		downloaderArgs = append(downloaderArgs, "-q", *quality)
	} else {
		downloaderArgs = append(downloaderArgs, "-q", "720p60")
	}
	if *start != "" {
		downloaderArgs = append(downloaderArgs, "-b", *start)
	}
	if *end != "" {
		downloaderArgs = append(downloaderArgs, "-e", *end)
	}

	cmd := exec.Command(twitchDownloaderCLIPath, downloaderArgs...)
	// Pipe the command output to the application standard output
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		fmt.Println("")
		fmt.Println("Could not run the video downloader:", err)
	}
	fmt.Println("")

	// Upload the video
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithCredentialsFile(os.Getenv("GOOGLE_CLOUD_CREDENTIALS_PATH")), option.WithScopes(drive.DriveFileScope))
	if err != nil {
		fmt.Println("Error creating Drive service:", err)
		return
	}

	// Open the local file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening the file:", err)
		return
	}
	defer file.Close()

	// Ensure folder
	parentFolderName := "twitch-videos"
	var parentFolderID string
	folderQuery := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder'", parentFolderName)
	existingFolders, err := srv.Files.List().Q(folderQuery).Do()
	if err != nil {
		fmt.Println("Error checking folder existence:", err)
		return
	}

	if len(existingFolders.Files) == 0 { // Create and set the folder
		folder := &drive.File{
			Name:     parentFolderName,
			MimeType: "application/vnd.google-apps.folder",
		}
		newFolder, err := srv.Files.Create(folder).Do()

		if err != nil {
			fmt.Println("Error creating folder:", err)
			return
		}

		fmt.Println("Parent folder created")
		parentFolderID = newFolder.Id
	} else { // Set the existing folder
		fmt.Println("Parent folder found")
		parentFolderID = existingFolders.Files[0].Id
	}

	// Create a new file in Google Drive
	fmt.Println("Uploading file...")
	newFile, err := srv.Files.Create(&drive.File{Name: fileName, Parents: []string{parentFolderID}}).Media(file).Do()
	if err != nil {
		fmt.Println("Error uploading file:", err)
		return
	}

	fmt.Println("File uploaded successfully. File ID:", newFile.Id)

	// Define the permission to be added (read-only access)
	permission := &drive.Permission{
		Type:         "user",
		Role:         "writer",
		EmailAddress: os.Getenv("SHARE_WITH_USER"),
	}

	// Add the permission to the file
	_, err = srv.Permissions.Create(newFile.Id, permission).Do()
	if err != nil {
		fmt.Println("Error adding permission:", err)
		return
	}

	fmt.Println("File shared successfully with", os.Getenv("SHARE_WITH_USER"))

	// Files cleanuo
	files, err := srv.Files.List().Q(fmt.Sprintf("'%s' in parents", parentFolderID)).Do()
	if err != nil {
		fmt.Println("Error listing files:", err)
		return
	}

	// Delete channel old files
	for _, file := range files.Files {
		if file.Id != newFile.Id && strings.HasPrefix(file.Name, videoData.UserName) {
			err := srv.Files.Delete(file.Id).Do()
			if err != nil {
				fmt.Println("Error deleting file:", file.Name, err)
			} else {
				fmt.Println("Old file deleted successfully:", file.Name)
			}
		}
	}
}

func loadEnvVariables() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	requiredVars := []string{
		"TWITCH_CLIENT_ID", "TWITCH_CLIENT_SECRET", "LOCAL_FILE_PATH",
		"TWITCH_DOWNLOADER_CLI_PATH", "GOOGLE_CLOUD_CREDENTIALS_PATH",
		"SHARE_WITH_USER",
	}

	for _, variable := range requiredVars {
		if os.Getenv(variable) == "" {
			fmt.Println("Missing environment variable:", variable)
			return
		}
	}
}
