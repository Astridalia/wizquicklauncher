package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
)

// WizardInfo represents the information for a single Wizard101 account.
type WizardInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
	XPos     int    `json:"xPos"`
	YPos     int    `json:"yPos"`
}

// Config represents the configuration options for the program.
type Config struct {
	FilePath     string       `json:"filePath"`
	AccountsData []WizardInfo `json:"accountsData"`
}

// Application represents the main application.
type Application struct {
	Config        *Config
	AccountsArray []WizardInfo
}

var (
	user32         = windows.NewLazySystemDLL("user32.dll")
	setWindowPos   = user32.NewProc("SetWindowPos")
	getClassNameW  = user32.NewProc("GetClassNameW")
	postMessageW   = user32.NewProc("PostMessageW")
	setWindowTextW = user32.NewProc("SetWindowTextW")
	enumWindows    = user32.NewProc("EnumWindows")
)

// NewApplication creates a new instance of the Application.
func NewApplication(config *Config) *Application {
	return &Application{
		Config: config,
	}
}

// moveWindow moves the specified window to the given position (x, y).
func moveWindow(handle windows.Handle, x, y int) {
	setWindowPos.Call(
		uintptr(handle),
		0,
		uintptr(x),
		uintptr(y),
		0,
		0,
		0x0001,
	)
}

// getAllWizardHandles returns a map of all Wizard handles.
func getAllWizardHandles() map[windows.Handle]struct{} {
	targetClass := "Wizard Graphical Client"
	handles := make(map[windows.Handle]struct{})

	enumWindowsCallback := syscall.NewCallback(func(hwnd windows.Handle, lparam uintptr) uintptr {
		classNameBuf := make([]uint16, 256)
		getClassNameW.Call(
			uintptr(hwnd),
			uintptr(unsafe.Pointer(&classNameBuf[0])),
			256,
		)
		className := windows.UTF16ToString(classNameBuf)

		if className == targetClass {
			handles[hwnd] = struct{}{}
		}

		return 1
	})

	enumWindows.Call(enumWindowsCallback, 0)
	return handles
}

// openWizard opens the Wizard101 application and returns the process.
func (app *Application) openWizard(filepath string) (*os.Process, error) {
	command := exec.Command("cmd", "/C", "cd", filepath, "&&", "start", "WizardGraphicalClient.exe", "-L", "login.us.wizard101.com", "12000")
	if err := command.Start(); err != nil {
		return nil, err
	}
	return command.Process, nil
}

// sendChars sends characters to the specified window.
func sendChars(windowHandle windows.Handle, chars string) {
	for _, char := range chars {
		postMessageW.Call(
			uintptr(windowHandle),
			0x102,
			uintptr(char),
			0,
		)
	}
}

// wizardLogin performs login in the Wizard101 window.
func wizardLogin(windowHandle windows.Handle, username, password string) {
	sendChars(windowHandle, username)
	sendChars(windowHandle, "\t") // tab
	sendChars(windowHandle, password)
	sendChars(windowHandle, "\r") // enter

	// Set title
	setWindowTextW.Call(
		uintptr(windowHandle),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(fmt.Sprintf("[%s] Wizard101", username)))),
	)
}

// Run runs the main application logic.
func (app *Application) Run() {
	target := len(app.Config.AccountsData)
	initialHandles := getAllWizardHandles()
	initialHandlesLen := len(initialHandles)

	for i := 0; i < target; i++ {
		process, err := app.openWizard(app.Config.FilePath)
		if err != nil {
			fmt.Println("Failed to open Wizard101 process:", err)
			continue
		}
		defer process.Release()
	}

	// Wait a little
	time.Sleep(2 * time.Second)

	var handles map[windows.Handle]struct{}
	for len(handles) != target+initialHandlesLen {
		handles = getAllWizardHandles()
		time.Sleep(500 * time.Millisecond)
	}

	newHandles := make(map[windows.Handle]struct{})
	for handle := range handles {
		if _, ok := initialHandles[handle]; !ok {
			newHandles[handle] = struct{}{}
		}
	}

	i := 0
	for handle := range newHandles {
		wizardLogin(handle, app.Config.AccountsData[i].Username, app.Config.AccountsData[i].Password)
		moveWindow(handle, app.Config.AccountsData[i].XPos, app.Config.AccountsData[i].YPos)
		i++
	}
}

func main() {
	var configPath string

	app := &cli.App{
		Name:  "Wizard101 Automation",
		Usage: "Automates login to Wizard101 accounts",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config, c",
				Value:       "config.json",
				Usage:       "Path to the configuration file",
				Destination: &configPath,
			},
		},
		Action: func(c *cli.Context) error {
			viper.SetConfigFile(configPath)
			if err := viper.ReadInConfig(); err != nil {
				return err
			}

			var config Config
			if err := viper.Unmarshal(&config); err != nil {
				return err
			}

			// Initialize the Application struct with the accounts data from the configuration.
			app := NewApplication(&config)

			fxApp := fx.New(
				fx.Provide(func() *Config { return &config }),
				fx.Provide(func() *Application { return app }), // Provide the Application instance for injection.
				fx.Invoke(func(app *Application) {
					app.Run() // Call the Run method inside the fx.Invoke function.
				}),
			)

			if err := fxApp.Start(context.Background()); err != nil {
				return err
			}

			// Run the application
			if err := fxApp.Err(); err != nil {
				return err
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
