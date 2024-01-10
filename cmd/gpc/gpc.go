package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	pc "github.com/steeringwaves/git-project-creator"
)

var (
	projectCreator pc.ProjectCreator
	data           string
)

func main() {
	var rootCmd = &cobra.Command{Use: "gpc"}

	projectCreator.Prompt = true

	rootCmd.PersistentFlags().StringVarP(&projectCreator.GitRepository.URL, "repo", "r", "", "Git repository URL to clone")
	rootCmd.PersistentFlags().StringVarP(&projectCreator.GitRepository.Branch, "branch", "b", "", "Git branch to clone")
	rootCmd.PersistentFlags().StringVarP(&projectCreator.GitRepository.Tag, "tag", "t", "", "Git tag to clone")
	rootCmd.PersistentFlags().StringVarP(&projectCreator.ExistingDir, "existing", "e", "", "Existing directory to be used as a template")
	rootCmd.PersistentFlags().StringVarP(&projectCreator.DownloadURL, "download", "u", "", "URL to download the template from (e.g., GitHub releases, support tar, tar.gz, tar.xz, tar.bz2, zip)")
	rootCmd.PersistentFlags().StringVarP(&data, "data", "D", "", "Data for template in JSON or YAML format for templating")
	rootCmd.PersistentFlags().StringVarP(&projectCreator.DestDir, "dir", "d", "", "Destination directory for the new project")
	rootCmd.PersistentFlags().BoolVarP(&projectCreator.Overwrite, "overwrite", "o", false, "Overwrite existing directory if it exists")

	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		if projectCreator.GitRepository.URL == "" && projectCreator.ExistingDir == "" && projectCreator.DownloadURL == "" {
			cmd.Usage()
			os.Exit(1)
			return
		}

		// Create a project based on the provided configuration
		err := projectCreator.CreateProject(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error creating the project:", err)
			os.Exit(1)
			return
		}

		fmt.Println("Project successfully created!")
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
