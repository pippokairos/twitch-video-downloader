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

const (
	DefaultQuality   = "720p60"
	ParentFolderName = "twitch-videos"
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

	client := &http.Client{}

	accessToken, err := getAccessToken()
	if err != nil {
		fmt.Println("Error on getting the access token:", err)
		return
	}

	UsersResponse, err := getUsers(client, accessToken, *name)
	if err != nil {
		fmt.Println("Error on getting the users response:", err)
		return
	}

	VideosResponse, err := getVideos(client, accessToken, UsersResponse)
	if err != nil {
		fmt.Println("Error on getting the videos response:", err)
		return
	}

	err = downloadVideo(VideosResponse, *quality, *start, *end)
	if err != nil {
		fmt.Println("Error on downloading the video:", err)
		return
	}

	err = uploadVideo(VideosResponse)
	if err != nil {
		fmt.Println("Error on getting the users response:", err)
		return
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

func getAccessToken() (string, error) {
	// https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/#client-credentials-flow-example
	authData := url.Values{}
	authData.Set("client_id", os.Getenv("TWITCH_CLIENT_ID"))
	authData.Set("client_secret", os.Getenv("TWITCH_CLIENT_SECRET"))
	authData.Set("grant_type", "client_credentials")

	res, err := http.PostForm("https://id.twitch.tv/oauth2/token", authData)
	if err != nil {
		return "", fmt.Errorf("Error making request: %v", err)
	}
	defer res.Body.Close()

	// Read the response body into a slice of bytes
	authBodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading auth response body: %v", err)
	}

	// Parse the JSON response
	var tokenResponse TokenResponse
	err = json.Unmarshal(authBodyBytes, &tokenResponse)
	if err != nil {
		return "", fmt.Errorf("Error decoding auth response body to JSON: %v", err)
	}

	return tokenResponse.AccessToken, nil
}

func getUsers(client *http.Client, accessToken string, name string) (UsersResponse, error) {
	// https://dev.twitch.tv/docs/api/reference/#get-users
	var UsersResponse UsersResponse
	userUrl := "https://api.twitch.tv/helix/users?login=" + name
	req, err := http.NewRequest("GET", userUrl, nil)
	if err != nil {
		return UsersResponse, fmt.Errorf("Error creating the user GET request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Client-Id", os.Getenv("TWITCH_CLIENT_ID"))
	res, err := client.Do(req)
	if err != nil {
		return UsersResponse, fmt.Errorf("Error making the user http request: %v", err)
	}

	// Read the response body into a slice of bytes
	usersBodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return UsersResponse, fmt.Errorf("Error reading users response body: %v", err)
	}

	// Parse the JSON response
	err = json.Unmarshal(usersBodyBytes, &UsersResponse)
	if err != nil {
		return UsersResponse, fmt.Errorf("Error decoding users response body to JSON: %v", err)
	}

	return UsersResponse, nil
}

func getVideos(client *http.Client, accessToken string, UsersResponse UsersResponse) (VideosResponse, error) {
	// https://dev.twitch.tv/docs/api/reference/#get-videos
	var VideosResponse VideosResponse
	videosUrl := "https://api.twitch.tv/helix/videos?type=archive&sort=time&first=1&user_id=" + UsersResponse.Data[0].Id
	req, err := http.NewRequest("GET", videosUrl, nil)
	if err != nil {
		return VideosResponse, fmt.Errorf("Error creating the videos GET request: %v", err)
	}

	// TODO: Make func to be DRYer
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Client-Id", os.Getenv("TWITCH_CLIENT_ID"))
	res, err := client.Do(req)
	if err != nil {
		return VideosResponse, fmt.Errorf("Error making the videos http request: %v", err)
	}

	// Read the response body into a slice of bytes
	videosBodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return VideosResponse, fmt.Errorf("Error reading videos response body: %v", err)
	}

	err = json.Unmarshal(videosBodyBytes, &VideosResponse)
	if err != nil {
		return VideosResponse, fmt.Errorf("Error decoding videos response body to JSON: %v", err)
	}

	return VideosResponse, nil
}

func fileName(videoData VideosData) string {
	return videoData.UserName + " - " + videoData.Title + ".mp4"
}

func filePath(videoData VideosData) string {
	return os.Getenv("LOCAL_FILE_PATH") + fileName(videoData)
}

func downloadVideo(VideosResponse VideosResponse, quality string, start string, end string) error {
	twitchDownloaderCLIPath := os.Getenv("TWITCH_DOWNLOADER_CLI_PATH")
	downloaderArgs := getDownloaderArgs(VideosResponse.Data[0], quality, start, end)
	cmd := exec.Command(twitchDownloaderCLIPath, downloaderArgs...)
	// Pipe the command output to the application standard output
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		fmt.Println("")
		return fmt.Errorf("Could not run the video downloader: %v", err)
	}
	fmt.Println("")

	return nil
}

func getDownloaderArgs(videoData VideosData, quality string, start string, end string) []string {
	downloaderArgs := []string{"videodownload", "--id", videoData.ID, "-o", filePath(videoData)}

	if quality != "" {
		downloaderArgs = append(downloaderArgs, "-q", quality)
	} else {
		downloaderArgs = append(downloaderArgs, "-q", DefaultQuality)
	}
	if start != "" {
		downloaderArgs = append(downloaderArgs, "-b", start)
	}
	if end != "" {
		downloaderArgs = append(downloaderArgs, "-e", end)
	}

	return downloaderArgs
}

func uploadVideo(VideosResponse VideosResponse) error {
	ctx := context.Background()
	DriveService, err := drive.NewService(ctx, option.WithCredentialsFile(os.Getenv("GOOGLE_CLOUD_CREDENTIALS_PATH")), option.WithScopes(drive.DriveFileScope))
	if err != nil {
		return fmt.Errorf("Error creating Drive service: %v", err)
	}

	// Open the local file
	videoData := VideosResponse.Data[0]
	file, err := os.Open(filePath(videoData))
	if err != nil {
		return fmt.Errorf("Error opening the file: %v", err)
	}
	defer file.Close()

	// Ensure folder
	var parentFolderID string
	folderQuery := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder'", ParentFolderName)
	existingFolders, err := DriveService.Files.List().Q(folderQuery).Do()
	if err != nil {
		return fmt.Errorf("Error checking folder existence: %v", err)
	}

	if len(existingFolders.Files) == 0 { // Create and set the folder
		folder := &drive.File{
			Name:     ParentFolderName,
			MimeType: "application/vnd.google-apps.folder",
		}
		newFolder, err := DriveService.Files.Create(folder).Do()

		if err != nil {
			return fmt.Errorf("Error creating folder: %v", err)
		}

		fmt.Println("Parent folder created")
		parentFolderID = newFolder.Id
	} else { // Set the existing folder
		fmt.Println("Parent folder found")
		parentFolderID = existingFolders.Files[0].Id
	}

	// Create a new file in Google Drive
	fmt.Println("Uploading file...")
	newFile, err := DriveService.Files.Create(&drive.File{Name: fileName(videoData), Parents: []string{parentFolderID}}).Media(file).Do()
	if err != nil {
		return fmt.Errorf("Error uploading file: %v", err)
	}

	fmt.Println("File uploaded successfully. File ID:", newFile.Id)

	// Define the permission to be added (read-only access)
	permission := &drive.Permission{
		Type:         "user",
		Role:         "writer",
		EmailAddress: os.Getenv("SHARE_WITH_USER"),
	}

	// Add the permission to the file
	_, err = DriveService.Permissions.Create(newFile.Id, permission).Do()
	if err != nil {
		return fmt.Errorf("Error adding permission: %v", err)
	}

	fmt.Println("File shared successfully with", os.Getenv("SHARE_WITH_USER"))

	// Files cleanup
	files, err := DriveService.Files.List().Q(fmt.Sprintf("'%s' in parents", parentFolderID)).Do()
	if err != nil {
		return fmt.Errorf("Error listing files: %v", err)
	}

	// Delete channel old files
	for _, file := range files.Files {
		if file.Id != newFile.Id && strings.HasPrefix(file.Name, videoData.UserName) {
			err := DriveService.Files.Delete(file.Id).Do()
			if err != nil {
				return fmt.Errorf("Error deleting file: %v", err)
			} else {
				fmt.Println("Old file deleted successfully:", file.Name)
			}
		}
	}

	return nil
}
