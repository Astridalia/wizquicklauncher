# WizQuickLauncher

The WizQuickLauncher application is a command-line tool that automates the login process for multiple Wizard101 accounts. It allows you to log in to multiple accounts simultaneously and position each game window on the screen as desired.

### Prerequisites
Before running the application, ensure you have the following installed:

Go programming language (https://golang.org/dl/)

### Installation
1. Clone the repository or download the source code.

2. Build the application using the following command:
```bash
go build
```

### Usage
The WizQuickLauncher application requires a configuration file in JSON format to specify the accounts and their login details. The configuration file should be provided as a command-line flag.

Configuration File Format
The configuration file should have the following format:
```json
{
  "filePath": "C:\\Path\\To\\Wizard101\\Bin", 
  "accountsData": [
    {
      "username": "user1",
      "password": "pass1",
      "xPos": 100,
      "yPos": 200
    },
    {
      "username": "user2",
      "password": "pass2",
      "xPos": 300,
      "yPos": 400
    }
  ]
}
```
"FilePath": The file path to the Wizard101 directory. Make sure to use double backslashes (\\) in the path.

"AccountsData": An array containing login details for each account. Each account should be represented as an array with four elements:

"username": The username of the Wizard101 account.
"password": The password of the Wizard101 account.
"x": The x-coordinate position of the game window on the screen.
"y": The y-coordinate position of the game window on the screen.
Running the Application
To run the application, use the following command:
```bash
./WizQuickLauncher -config path/to/config.json
```

Replace path/to/config.json with the actual path to your configuration file.

The application will start logging in to the specified accounts and positioning the game windows on the screen according to the provided coordinates.

### Example Configuration
```bash
{
  "FilePath": "C:\\Users\\admin\\AppData\\Roaming\\Wizard101\\Bin",
  "AccountsData": [
    ["wizard1@example.com", "password123", "100", "100"],
    ["wizard2@example.com", "mypassword", "500", "200"],
    ["wizard3@example.com", "securepass", "300", "300"]
  ]
}
```
