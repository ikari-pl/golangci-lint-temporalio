# Temporal.io Go Linter

A linter for validating Temporal.io workflow and activity invocations in Go projects.

## Description

This linter ensures that calls to Temporal.io workflows and activities are made with the correct types and number of
arguments. It also checks that all arguments and return values are serializable, which is a requirement for Temporal.io
to be able to pass them to the workflow, or activity. Otherwise they assume the zero value of their types, making
it hard to find the bug.

## Features

* Checks for correct argument types and counts in workflow and activity calls.
* Validates that all fields in structs passed to workflows and activities are exported and serializable.  
* Supports variadic arguments in workflow and activity calls.

## Installation

To install the linter, you need to have Go installed on your system. You can then use `go get` to install the linter:

 ```bash
 go get github.com/ikari-pl/golangci-lint-temporalio
```

## Usage

To run the linter as a standalone command, navigate to the root directory of your Go project and run:

```bash
go vet -vettool=$(which golangci-lint-temporalio) PATH_TO_YOUR_PACKAGE
```

Replace temporalio-linter with the actual binary name if it differs.

## Contributing

Contributions to the linter are welcome. Please feel free to open issues or submit pull requests on the project's GitHub
repository.

---

Please note that the installation command assumes that the linter is hosted on GitHub at the provided path and that the
binary is named `golangci-lint-temporalio`. Adjust these details as necessary for the actual
project.                            


──────────────────

Note: This project is in the early stages of development and is not yet integrated into the golangci-lint suite.
Future updates will provide instructions for integration and usage within the golangci-lint framework.
