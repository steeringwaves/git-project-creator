package gpc

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/steeringwaves/git-project-creator/gorecurcopy"
	"gopkg.in/yaml.v2"
)

// Variables stores the variables to be used in the template
type Variables struct {
	Name        string      `yaml:"name"`        // Name of the variable
	Description string      `yaml:"description"` // Description of the variable
	Default     interface{} `yaml:"default"`     // Default value of the variable
}

// TemplateConfig stores the configuration from the .gpc.yml or .gpc.yaml file
type TemplateConfig struct {
	Templates []string    `yaml:"templates"` // List of template files
	Variables []Variables `yaml:"variables"` // List of variables
}

type GitRepository struct {
	URL    string
	Branch string
	Tag    string
	Commit string
}

type ProjectCreator struct {
	Prompt        bool          // Prompt flag indicates whether to prompt the user for input
	GitRepository GitRepository // Git repository to be used as a template
	ExistingDir   string        // Existing directory to be used as a template
	DownloadURL   string        // URL to download the template from (e.g. GitHub releases, support tar, tar.gz, tar.xz, tar.bz2, zip)

	Data      map[string]interface{} // Data to be injected into the template
	DestDir   string                 // Destination directory for the new project
	Overwrite bool                   // Overwrite flag indicates whether to overwrite existing directory
}

// CreateProject fetches the template and processes the files as templates
func (pc *ProjectCreator) CreateProject(data string) error {
	// Check if the destination directory exists
	if _, err := os.Stat(pc.DestDir); err == nil {
		// If the destination directory exists and overwrite flag is not set, prompt the user for confirmation
		if !pc.Overwrite {
			if pc.Prompt {
				fmt.Printf("Directory %s already exists. Do you want to overwrite it? (y/n): ", pc.DestDir)
				var input string
				fmt.Scanln(&input)

				if input != "y" && input != "Y" {
					return fmt.Errorf("directory already exists")
				}
			} else {
				return fmt.Errorf("directory already exists")
			}
		}
	}

	cliData, err := parseData(data)
	if err != nil {
		return err
	}

	// Fetch the template
	err = pc.FetchTemplate()
	if err != nil {
		return err
	}

	// Read the template configuration
	config, err := readTemplateConfig(pc.DestDir)
	if err != nil {
		return err
	}

	pc.Data = make(map[string]interface{})

	// Loop through the variables and prompt the user for input
	for _, variable := range config.Variables {
		// Check if the variable is already provided in the data
		val, ok := cliData[variable.Name]
		if ok {
			pc.Data[variable.Name] = val
			continue
		}

		if pc.Prompt {
			// Prompt the user for input
			fmt.Printf("%s (%v): ", variable.Description, variable.Default)
			var input string
			fmt.Scanln(&input)

			// If the user input is empty, use the default value
			if input == "" {
				pc.Data[variable.Name] = variable.Default
			} else {
				pc.Data[variable.Name] = input
			}
		} else {
			pc.Data[variable.Name] = variable.Default
		}
	}

	// Process files in the destination directory as templates
	err = processFilesAsTemplates(pc.DestDir, pc.Data, config)
	if err != nil {
		return err
	}

	return nil
}

// Function to fecth the template
func (pc *ProjectCreator) FetchTemplate() error {
	if pc.GitRepository.URL != "" {
		return pc.cloneRepository()
	}

	if pc.ExistingDir != "" {
		return pc.copyExistingDir()
	}

	if pc.DownloadURL != "" {
		return pc.downloadAndExtract()
	}

	return fmt.Errorf("no template source provided")
}

// Function to copy an existing directory
func (pc *ProjectCreator) copyExistingDir() error {
	// check if the existing directory exists
	if _, err := os.Stat(pc.ExistingDir); err != nil {
		return err
	}

	// copy the existing directory to the destination directory
	// return exec.Command("cp", "-r", pc.ExistingDir, pc.DestDir).Run() // TODO use Go's copy function
	return gorecurcopy.CopyDirectory(pc.ExistingDir, pc.DestDir, []string{".git"})
}

// Function to clone a Git repository
func (pc *ProjectCreator) cloneRepository() error {
	// build the git clone command
	commands := []string{"git", "clone"}

	if pc.GitRepository.Branch != "" {
		commands = append(commands, "--branch", pc.GitRepository.Branch)
	} else if pc.GitRepository.Tag != "" {
		commands = append(commands, "--branch", pc.GitRepository.Tag)
	} else if pc.GitRepository.Commit != "" {
		commands = append(commands, "--branch", pc.GitRepository.Commit)
	}

	commands = append(commands, pc.GitRepository.URL, pc.DestDir)

	// execute the git clone command
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Function to download a file from a URL and extract it to a directory
func (pc *ProjectCreator) downloadAndExtract() error {
	// create a temporary directory to download the template
	tempDir, err := ioutil.TempDir("", "template")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tempDir)

	// download the template to the temporary directory using net/http
	resp, err := http.Get(pc.DownloadURL)
	if err != nil {
		return err
	}

	filename := filepath.Join(tempDir, "template")
	var extractCommands []string

	// check the type of file to be downloaded
	contentType := resp.Header.Get("Content-Type")
	switch contentType {
	case "application/x-gzip":
		filename += ".tar.gz"
		extractCommands = []string{"tar", "-xzf", filename, "-C", pc.DestDir, "--strip-components=1"}
	case "application/x-xz":
		filename += ".tar.xz"
		extractCommands = []string{"tar", "-xJf", filename, "-C", pc.DestDir, "--strip-components=1"}
	case "application/x-bzip2":
		filename += ".tar.bz2"
		extractCommands = []string{"tar", "-xjf", filename, "-C", pc.DestDir, "--strip-components=1"}
	case "application/zip":
		// filename += ".zip"
		// extractCommands = []string{"unzip", filename, "-d", pc.DestDir}
		return fmt.Errorf("unsupported file type: %s (zip does not support --strip-components, you should use a tarball)", contentType)
	default:
		return fmt.Errorf("unsupported file type: %s", contentType)
	}

	defer resp.Body.Close()

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// Create the destination directory
	err = os.MkdirAll(pc.DestDir, 0755)
	if err != nil {
		return err
	}

	// execute the extract command
	cmd := exec.Command(extractCommands[0], extractCommands[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Function to read template configuration from .gpc.yml or .gpc.yaml
func readTemplateConfig(directory string) (*TemplateConfig, error) {
	configFilePaths := []string{".gpc.yml", ".gpc.yaml"}

	for _, configFile := range configFilePaths {
		configPath := filepath.Join(directory, configFile)
		if _, err := os.Stat(configPath); err == nil {
			configData, err := ioutil.ReadFile(configPath)
			if err != nil {
				return nil, err
			}

			var config TemplateConfig
			err = yaml.Unmarshal(configData, &config)
			if err != nil {
				return nil, err
			}

			return &config, nil
		}
	}

	// If no template configuration file is found, return a default empty config
	return &TemplateConfig{}, nil
}

// Function to process files in a directory as templates
func processFilesAsTemplates(directory string, variables map[string]interface{}, config *TemplateConfig) error {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip processing the .git directory
		if info.IsDir() && strings.HasSuffix(path, "/.git") {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			// Check if the file should be ignored or templated based on the config
			shouldTemplate := false

			if len(config.Templates) > 0 {
				for _, templatePattern := range config.Templates {
					if match, _ := filepath.Match(templatePattern, info.Name()); match {
						shouldTemplate = true
						break
					}
				}
			}

			if shouldTemplate {
				// Read the file content
				content, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}

				// Create a template from the file content
				tmpl, err := template.New(info.Name()).Parse(string(content))
				if err != nil {
					return err
				}

				// Create a new file for templated content
				newFile, err := os.Create(path)
				if err != nil {
					return err
				}
				defer newFile.Close()

				// Execute the template with provided variables
				err = tmpl.Execute(newFile, variables)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return err
}

// Function to parse variables in JSON or YAML format
func parseData(varsStr string) (map[string]interface{}, error) {
	var varsMap map[string]interface{}
	err := json.Unmarshal([]byte(varsStr), &varsMap)
	if err != nil {
		err = yaml.Unmarshal([]byte(varsStr), &varsMap)
		if err != nil {
			return nil, err
		}
	}

	return varsMap, nil
}
