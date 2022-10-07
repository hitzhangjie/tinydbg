# Installation
The following instructions are known to work on Linux/amd64.

Clone the git repository and build:

```
$ git clone https://github.com/go-delve/delve
$ cd delve
$ go install github.com/go-delve/delve/cmd/dlv
```

On Go version 1.16 or later, this command will also work:

```
$ go install github.com/go-delve/delve/cmd/dlv@latest
```

See `go help install` for details on where the `dlv` executable is saved.
