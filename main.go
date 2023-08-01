package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/urfave/cli"
	"go.uber.org/fx"
)

var (
	user32         = windows.NewLazySystemDLL("user32.dll")
	setWindowPos   = user32.NewProc("SetWindowPos")
	getClassNameW  = user32.NewProc("GetClassNameW")
	postMessageW   = user32.NewProc("PostMessageW")
	setWindowTextW = user32.NewProc("SetWindowTextW")
	enumWindows    = user32.NewProc("EnumWindows")
)

// Config represents the configuration options for the program.
type Config struct {
	FilePath     string     `json:"FilePath"`
	AccountsData [][]string `json:"AccountsData"`
}

// Application represents the main application.
type Application struct {
	Config        *Config
	AccountsArray [][]string
}

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
		wizardLogin(handle, app.Config.AccountsData[i][0], app.Config.AccountsData[i][1])
		moveWindow(handle, atoi(app.Config.AccountsData[i][2]), atoi(app.Config.AccountsData[i][3]))
		i++
	}
}

func atoi(s string) int {
	num, _ := strconv.Atoi(s)
	return num
}

// readConfigFromFile reads the configuration from the specified JSON file.
func readConfigFromFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	var configPath string

	app := cli.NewApp()
	app.Name = "Wizard101 Automation"
	app.Usage = "Automates login to Wizard101 accounts"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config, c",
			Value:       "config.json",
			Usage:       "Path to the configuration file",
			Destination: &configPath,
		},
	}
	app.Action = func(c *cli.Context) error {
		config, err := readConfigFromFile(configPath)
		if err != nil {
			return err
		}

		// Initialize the Application struct with the accounts data from the configuration.
		app := NewApplication(config)
		app.AccountsArray = config.AccountsData

		fxApp := fx.New(
			fx.Provide(func() *Config { return config }),
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
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
