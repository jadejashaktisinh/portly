/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID     string `json:"id"`
		Email  string `json:"email"`
		APIKey string `json:"apiKey"`
	} `json:"user"`
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "login comand is used to authenticate the user and store the API key in the system keyring",
	Long:  `login command is used to authenticate the user and store the API key in the system keyring. It prompts the user for their email and password, sends a login request to the server, and if successful, stores the API key in the system keyring for future use.`,
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter your email: ")

		// 2. Read the input until the user presses Enter
		email, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}

		fmt.Print("Enter your password: ")
		password, err := reader.ReadString('\n')

		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}

		// 3. Clean up the trailing newline character
		email = strings.TrimSpace(email)
		password = strings.TrimSpace(password)

		// 4. Print the captured email and password (for demonstration purposes)
		fmt.Printf("Hello, %s!\n", email)
		fmt.Printf("Your password is: %s\n", password)

		login(email, password)

	},
}

func init() {
	portlyCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func login(email, password string) {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	serverUrl := os.Getenv("NODE_SERVER_URL")
	service := os.Getenv("CLI_SERVICE_NAME")
	user := os.Getenv("CLI_USER_NAME")
	res, err := http.Post(serverUrl+"/auth/login", "application/json", strings.NewReader(fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)))
	if err != nil {
		log.Fatal("Error sending login request:", err)
	}
	defer res.Body.Close()

	rawBody, err := io.ReadAll(res.Body)
	jsonResponse := LoginResponse{}
	err = json.Unmarshal(rawBody, &jsonResponse)
	if err != nil {
		log.Fatal("Error parsing JSON response:", err)
	}

	if res.StatusCode == http.StatusOK {
		fmt.Println("Login successful!", jsonResponse.User.APIKey)
		// 1. Store the secret string
		err := keyring.Set(service, user, jsonResponse.User.APIKey)
		if err != nil {
			log.Fatalf("Failed to set secret: %v", err)
		}
		fmt.Println("Secret stored successfully!")

	} else {
		fmt.Printf("Login failed with status code: %d %s\n", res.StatusCode, jsonResponse.User.APIKey)
	}
}
