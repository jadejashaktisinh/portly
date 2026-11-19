/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

type LoginResponse struct {
	Token string `json:"token"`
	Error string `json:"error"`
	User  struct {
		ID     string `json:"id"`
		Email  string `json:"email"`
		APIKey string `json:"apiKey"`
	} `json:"user"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

		login(email, password)

	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func login(email, password string) {
	serverURL := apiServerURL()
	service := keyringService()
	user := keyringUser()

	payload, err := json.Marshal(loginRequest{Email: email, Password: password})
	if err != nil {
		log.Fatal("Error encoding login request:", err)
	}

	res, err := http.Post(serverURL+"/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Fatal("Error sending login request:", err)
	}
	defer res.Body.Close()

	rawBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Error reading login response:", err)
	}

	jsonResponse := LoginResponse{}
	if err := json.Unmarshal(rawBody, &jsonResponse); err != nil {
		log.Fatal("Error parsing JSON response:", err)
	}

	if res.StatusCode != http.StatusOK {
		message := jsonResponse.Error
		if message == "" {
			message = http.StatusText(res.StatusCode)
		}
		fmt.Printf("Login failed: %s\n", message)
		os.Exit(1)
	}

	if jsonResponse.User.APIKey == "" {
		log.Fatal("Login response did not contain an API key")
	}

	if err := keyring.Set(service, user, jsonResponse.User.APIKey); err != nil {
		log.Fatalf("Failed to store API key: %v", err)
	}
	fmt.Println("Login successful. API key stored in the system keyring.")
}
