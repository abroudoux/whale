package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

//go:embed config.json
var configFile string
var config Config

type Config struct {
	Ui struct {
		CursorColor string `json:"cursorColor"`
		BranchColor string `json:"branchColor"`
		ContainerSelectedColor string `json:"containerSelectedColor"`
		ActionSelectedColor string `json:"actionSelectedColor"`
	} `json:"Ui"`
}

func main() {
	err := loadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if !isDockerInstalled() {
		println("Docker is not installed")
		os.Exit(1)
	}

	if !isDockerRunning() {
		println("Docker is not running")
		os.Exit(1)
	}

	containers, err := getContainers()
	if err != nil {
		println("Error getting containers")
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		flagMode(containers)
		os.Exit(0)
	}

	containerSelected, err := chooseContainer(containers)
	if err != nil {
		println("Error choosing container", err)
		os.Exit(1)
	}

	container := convertStringToContainer(containerSelected)

	debugPrintContainerInfos(container)

	actionSelected, err := chooseAction(container)
	if err != nil {
		println("Error choosing action", err)
		os.Exit(1)
	}

	err = doAction(actionSelected, container)
	if err != nil {
		println("Error doing action", err)
		os.Exit(1)
	}
}

func loadConfig() error {
	err := json.Unmarshal([]byte(configFile), &config)
	if err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	return nil
}

func isDockerInstalled() bool {
	cmd := exec.Command("docker", "-v")
	err := cmd.Run()
	return err == nil
}

func isDockerRunning() bool {
	cmd := exec.Command("docker", "container", "ls")
	err := cmd.Run()
	return err == nil
}

func flagMode(containers []string) {
	flag := os.Args[1]

	switch flag {
	case "--run", "-r":
		containerSelected, err := chooseContainer(containers)
		if err != nil {
			println("Error choosing container")
			os.Exit(1)
		}

		container := convertStringToContainer(containerSelected)

		actionSelected, err := chooseAction(container)
		if err != nil {
			println("Error choosing action")
			os.Exit(1)
		}

		println("Action selected: ", actionSelected)
	case "--help", "-h":
		printHelpManual()
	case "--version", "-v":
		fmt.Println("0.0.1")
	}
}

func printHelpManual() {
	fmt.Println("Usage: whale [options]")
	fmt.Printf("  %-20s %s\n", "whale [--run | -r]", "Run the program")
	fmt.Printf("  %-20s %s\n", "whale [--help | -h]", "Show this help message")
}

func getContainers() ([]string, error) {
	cmd := exec.Command("docker", "container", "ls", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var containers []string

	for i, line := range lines {
		if i == 0 {
			continue
		}

		containers = append(containers, line)
	}

	return containers, nil
}


type containerChoice struct {
	containers []string
	cursor    int
	selectedContainer string
}

func initialContainerModel(containers []string) containerChoice {
	return containerChoice{
		containers: containers,
		cursor:    len(containers) - 1,
		selectedContainer: "",
	}
}

func (menu containerChoice) Init() tea.Cmd {
	return nil
}

func (menu containerChoice) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return menu, tea.Quit
		case "up":
			menu.cursor--
			if menu.cursor < 0 {
				menu.cursor = len(menu.containers) - 1
			}
		case "down":
			menu.cursor++
			if menu.cursor >= len(menu.containers) {
				menu.cursor = 0
			}
		case "enter":
			menu.selectedContainer = menu.containers[menu.cursor]
			return menu, tea.Quit
		}
	}

	return menu, nil
}

func (menu containerChoice) View() string {
	s := "\033[H\033[2J"
    s += "Choose a container:\n\n"

	for i, container := range menu.containers {
        cursor := " "

        if menu.cursor == i {
            cursor = renderCursor()
            s += fmt.Sprintf("%s %s\n", cursor, renderContainerSelected(container, true))
            // s += fmt.Sprintf("%s %s\n", cursor, container)
        } else {
            s += fmt.Sprintf("%s %s\n", cursor, renderContainerSelected(container, false))
            // s += fmt.Sprintf("%s %s\n", cursor, container)
        }
    }

    return s
}

func chooseContainer(containers []string) (string, error) {
	containersMenu := tea.NewProgram(initialContainerModel(containers))
	finalModel, err := containersMenu.Run()
	if err != nil {
		return "", err
	}

	containerMenu := finalModel.(containerChoice)
	return containerMenu.selectedContainer, nil
}

func renderCursor() string {
	render := fmt.Sprintf("\033[%sm>\033[0m", config.Ui.CursorColor)
	return render
}

func renderContainerSelected(container string, isSelected bool) string {
    if isSelected {
		return fmt.Sprintf("\033[%sm%s\033[0m", config.Ui.ContainerSelectedColor, container)
    }
    return container
}


type actionChoice struct {
	actions []string
	cursor int
	selectedAction string
	selectedContainer Container
}

func initialActionModel(container Container) actionChoice {
	actions := []string{
		"Exit",
		"Copy container ID",
	}

	return actionChoice{
		actions: actions,
		cursor: len(actions) - 1,
		selectedAction: "",
		selectedContainer: container,
	}
}

func (menu actionChoice) Init() tea.Cmd {
	return nil
}

func (menu actionChoice) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return menu, tea.Quit
		case "up":
			menu.cursor--
			if menu.cursor < 0 {
				menu.cursor = len(menu.actions) - 1
			}
		case "down":
			menu.cursor++
			if menu.cursor >= len(menu.actions) {
				menu.cursor = 0
			}
		case "enter":
			menu.selectedAction = menu.actions[menu.cursor]
			return menu, tea.Quit
		}
	}

	return menu, nil
}

func (menu actionChoice) View() string {
	s := "\033[H\033[2J"
	s += fmt.Sprintf("Container: %s\n\n", menu.selectedContainer.Name)

	for i, action := range menu.actions {
		cursor := " "

		if menu.cursor == i {
			cursor = renderCursor()
			s += fmt.Sprintf("%s %s\n", cursor, renderActionSelected(action, true))
		} else {
			s += fmt.Sprintf("%s %s\n", cursor, renderActionSelected(action, false))
		}
	}

	return s
}

func renderActionSelected(action string, isSelected bool) string {
    if isSelected {
        return fmt.Sprintf("\033[%sm%s\033[0m", config.Ui.ActionSelectedColor, action)
    }
    return action
}

func chooseAction(container Container) (string, error) {
	actionsMenu := tea.NewProgram(initialActionModel(container))
	finalModel, err := actionsMenu.Run()
	if err != nil {
		return "", err
	}

	actionMenu := finalModel.(actionChoice)
	return actionMenu.selectedAction, nil
}

func doAction(action string, container Container) error {
	switch action {
	case "Exit":
		os.Exit(0)
	case "Copy container ID":
		containerID := container.ID
		err := copyContainerId(containerID)
		if err != nil {
			println(err)
			os.Exit(1)
		}
	}

	return nil
}

func copyContainerId(container string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(container)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error copying container ID: %v", err)
	}

	println("Container ID copied to clipboard")
	return nil
}

type Container struct {
	ID string
	Image string
	Command string
	Created string
	Status string
	Ports string
	Name string
}

func convertStringToContainer(str string) Container {
	fields := strings.Fields(str)
	id := fields[0]
	image := fields[1]
	command := fields[2]
	created := strings.Join(fields[3:6], " ")

	status := ""
	ports := ""
	if len(fields) > 10 {
		status = strings.Join(fields[6:10], " ")
	}

	name := fields[len(fields)-1]
	return Container{
		ID:      id,
		Image:   image,
		Command: command,
		Created: created,
		Status:  status,
		Ports:   ports,
		Name:    name,
	}
}

func debugPrintContainerInfos(container Container) {
	println("Container id: ", container.ID)
	println("Container image: ", container.Image)
	println("Container command: ", container.Command)
	println("Container created: ", container.Created)
	println("Container status: ", container.Status)
	println("Container ports: ", container.Ports)
	println("Container name: ", container.Name)
}