# cnb2cf
A tool to convert [Cloud Native Buildpacks](https://buildpacks.io/) (CNBs) to a single Cloud Foundry Buildpack (Shimmed buildpack)

A _shimmed buildpack_ is a wrapper around a group of CNBs, allowing them to run on Cloud Foundry deployments.

## Installation

### Requirements
- Go 1.11+

```
$ git clone https://github.com/cloudfoundry/cnb2cf
$ cd cnb2cf
$ ./scripts/build.sh
```

The binary (`cnb2cf`) can be found in the `build` dir.

## Usage

`cnb2cf package -stack <stack> [-cached] [-version <version>] [-cachedir <path to cachedir>] [-manifestpath <optional path to manifest>]`

This command creates a shimmed buildpack `.zip` file when run from within a shimmed buildpacks root directory. This allows you to cache the CNB dependencies in your shimmed buildpack, and to be run as a github url. The command must be run from the directory of a shimmed buildpack repo.

An example of the shimmed buildpack `buildpack.toml` can be found [here](https://github.com/cloudfoundry/cnb2cf/blob/44c3288c816570b162bdb7fa1a3f69c87603eb67/integration/testdata/metabuildpack_lc_0.7.x/buildpack.toml). It must have the lifecycle as a dependency along with other required dependencies. 

The output of the command is a buildpack `.zip` file in the current directory, with the name `<language>_buildpack[-<cached>]-<stack>-<version>.zip`. That zip file can be then uploaded to Cloud Foundry by running the `cf create-buildpack` command.

## Simple Workflow Example

A simple example workflow using the using a shimmed python Cloud Native Buildpack:

```
$ cd <directory-that-contains-the-shimmed-buildpack.toml>

$ cnb2cf package -stack cflinuxfs3 -version 1.0.0
# the above produces python_buildpack-cflinuxfs3-1.0.0.zip

# then upload to Cloud Foundry using the cf cli
$ cf create-buildpack my_shimmed_buildpack python_buildpack-cflinuxfs3-1.0.0.zip 1
```

## Debug Options

For additional output during detection phase set the CF environment variable
`LOG_LEVEL` to `debug`, using `cf set-env` Additional `LOG_LEVEL` options are
specified
[here](https://github.com/apex/log/blob/baa5455d10123171ef1951381610c51ad618542a/levels.go#L25)

