# Git Project Creator (gpc)

![workflow](https://github.com/steeringwaves/git-project-creator/actions/workflows/test.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/steeringwaves/git-project-creator.svg)](https://pkg.go.dev/github.com/steeringwaves/git-project-creator)

gpc is a command-line application for creating projects based on templates.
It supports cloning from Git repositories, using existing directories as templates, and downloading templates from URLs (tar.gz/tar.gz2/tar.xz).
If the repo contains either `.gpc.yml` or `gpc.yaml` then the file is parsed to provide the needed information for templating this project.

## Installation

Ensure that you have a supported version of Go properly installed and setup. You can find the minimum required version of Go in the go.mod file.

You can then install the latest release globally by running:

```sh
go install github.com/steeringwaves/git-project-creator/v1/cmd/gpc@latest
```

Or you can install into another directory:

```sh
env GOBIN=/bin go install github.com/steeringwaves/git-project-creator/v1/cmd/gpc@latest
```

## Example configuration file

```yaml
templates:
  - README.md
variables:
  - name: name
    description: Name of the project
    default: my-project
  - name: description
    description: Description of the project
    default: My project
  - name: server
    description: Server the repo is hosted on
    default: github.com
  - name: org
    description: License of the project
    default: steeringwaves

```

This configuration is used to tell gpc which files it should attempt to template. The variables object tells gpc the name of all the variables used for the templating process, as well as a description for each variable that is used to prompt the user for input.

## Command line options

```txt
Usage:
  gpc [flags]
Flags:
  -d, --dir string       Destination directory for the new project (required)
  -r, --repo string      Git repository URL to clone (optional)
  -b, --branch string    Git branch to clone (optional)
  -t, --tag string       Git tag to clone (optional)
  -e, --existing string  Existing directory to be used as a template
  -u, --download string  URL to download the template from (e.g., GitHub releases, support tar, tar.gz, tar.xz, tar.bz2, zip)
  -o, --overwrite        Overwrite existing directory if it exists (default is to prompt user if existing directory is found)
  -D, --data string      Data for template in JSON or YAML format for templating (default is to prompt user for input)
```

## Examples

```sh
# create new project in example/ from a git repo
gpc -d example -r https://github.com/steeringwaves/go-github-template.git

# create a new project in example/ from an existing directory and pass it all needed variables
gpc -d example -e ${HOME}/git/steeringwaves/go-github-template --data '{"name":"test","description":"test project","server":"github.com","org":"steeringwaves"}'

# create a new project in example/ from a tarball and pass it the name
gpc -d example -u https://github.com/steeringwaves/go-github-template/archive/refs/tags/v1.0.0.tar.gz --data '{"name":"test"}'
```
