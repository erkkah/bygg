![bygg! logo](icon.png)

# bygg! - a tiny build tool for portable projects

**bygg!** is a tiny build tool for maintaining portable builds for golang supported environments. It is similar to `make`, but much simpler and is guaranteed to work the same everywhere.

More highlights:

* Nothing to download and install
* Does not depend on the shell for a little bit of scripting
* Can download and validate the checksum of external dependencies
* Automatically rebuilds targets on dependency changes using watch mode
* Works great with CGO projects

## Getting started :rocket:

This is a silly "Hello, world" `byggfil` for a tiny project that does not need a build system:

```bygg
# Hello bygg file

<< Building "Hello, world" on ${GOOS}

all: hello

hello: hello.go
hello <- go build hello.go
```

To start the build, run:

```
$ go run github.com/erkkah/bygg
```

## The tool :hammer:

The `bygg` tool uses concepts similar to `make`, where the build process is described in a `byggfil` by listing dependencies and build steps. The `byggfil` is preprocessed as a `go` text template, with a couple of help functions.

This is obviously not `make`, and admittedly `go` templates are a bit weird, but it works really well for my needs and I've been able to simplify my build process, so I'm happy!

## Running a build

A build is started by running the tool in a directory containing a `byggfil`.
Since `bygg` is a no-dependencies tool, running using `go run` is fast enough:

```
$ go run github.com/erkkah/bygg
```

You can of course also install the tool to get faster startup times:

```
$ go get -u github.com/erkkah/bygg
```

To reduce friction even further, builds can be started with `go generate` by adding a simple `autoexec.go` file at the root of the project:

```go
// bygg! entry point, just run "go generate" to build

//go:generate go run github.com/erkkah/bygg

package main
```

The `bygg` tool accepts these arguments:

```
Usage:
  bygg <options> <target>

Options:
  -C string
      Base dir (default ".")
  -f string
      Bygg file (default "byggfil")
  -n  Performs a dry run
  -w  Watch mode
  -v  Verbose
  -vv Very verbose
```

The default target is "all".

### Watch mode

In watch mode, `bygg` will perform an initial build, and then wait for changes.
When changes are detected, it will automatically build outdated targets.

> Note that watch mode kicks in after template execution.

### Version check

Since v0.6.0 `bygg` supports verification of the tool version in the `byggfil`:

```
## bygg: ^0.6.0
```

The version requirement line must be the first line of the file and supports the version check prefixes `=`, `~` and `^`. In contrast with other tools there is no special treatment of `0.x.x` versions.

## `byggfil` syntax

### Dependencies

*Dependencies* are specified using colon - statements:

```
target: dependency1 dependency2
```

Existing targets will only be rebuilt if any of the depencies are newer, as usual.
For targets that should always be built, or when the dependency analysis is done by the build tool, building can be forced by prefixing the (possibly empty) dependency list with an exclamation mark:

```
target: !
```

### Build commands

*Build commands* for a target are specified using arrow statements:

```
target <- go build .
```

Multiple build commands for a single target are run in the order they appear.
Except for the special cases described below, build commands are external binaries that are run in the currently set environment.

#### Child builds

A child build can be run by using the internal `bygg` command:

```
target <- bygg -C ./path/to/submodule
```

There is nothing stopping you from running endless build loops using child builds. Have fun with that!

#### Downloads

If a build command starts with a URL to a `tar`, `tar.gz` or `tgz` file, that file will be downloaded and unpacked into a directory with the name of the target. The download can optionally be verified by an `md5` checksum:

```
lib <- https://where.files.live/mylittle.lib.tgz md5:f8288a861db7c97dc4750020c7c7aa6f
```

> NOTE: Downloads are considered to be up to date if the target directory is not older than the "Last-Modified" header sent from the server.

#### Removing files or directories

Build commands of the format `clean:path/to/file/or/dir` will remove the file/dir at the specified path.
To delete non-empty directories, the argument `-r` has to be specified.

#### Creating directories

Build commands of the format `mkdir:path/to/dir` will create the specified dir and all required dirs in the path.
If the dir already exists, nothing happens. If no path is given, the target will be used.

#### Copying files

Build commands of the format `copy:path/to/file` will copy the specified file to the target.

#### Logging

The special build operator `<<` prints the rest of the line to stdout:

```
parser-gen << Building the parser generator now!
```

The `<<` operator can also be used by itself, outside of build commands. In this case the output will be printed while interpreting, before target build commands are run.

```
<< Starting build {{date "2006-01-02 15:04:05"}}
```

### Variables

Variables are set and added to like this:

```
CFLAGS = -O2
CFLAGS += -Wall
```

Using `+=` to add to a variable will put a space between the old value and the addition.
This can be avoided using variable interpolation:

```
env.PATH = ${env.PATH}:/opt/frink/bin
```

Environment variables live in the `env` namespace:

```
env.CC = gcc
```

Variable interpolation uses familiar `$` - syntax, with or without curly brackes, and works in both lvalues and rvalues:

```
${MY_TARGET}: dependency $OBJFILES
```

#### Built-in variables

In addition to environment variables, the following variables are available:

* GOOS
* GOARCH
* GOVERSION

### Template execution

Before the build script is interpreted, it is run through the `go` [text template engine.](https://golang.org/pkg/text/template/)
This makes it possible to do things like:

```
REV = dev

{{if (exec "git" "status" "--porcelain") | eq "" }}
    repoState = Clean repo, tagging with git rev
    REV = {{slice (exec "git" "rev-parse" "HEAD") 0 7 }}
{{else}}
    repoState = Dirty repo, tagging as dev build
{{end}}

<< $repoState
```

The template data object is a map containing the environment and current `go` version:

```
<< Running in go version {{.GOVERSION}}
<< PATH is set to {{.env.PATH}}
```

In addition to the [standard functions](https://golang.org/pkg/text/template/#hdr-Functions), `bygg` adds the following:

#### env
Without arguments, returns the environment map. Equivalent to the `.env` data object field, except that it works in nested templates
as well.

With one argument, retrieves the given environment variable:

```
{{env "PATH"}}
```

With two arguments, sets the specified environment variable:

```
{{env "PATH" "/a/new/path"}}
```

#### exec
Returns the output of running the command specified by the first argument, with the rest of the arguments as command line arguments.

#### mustexec
Same as `exec`, but exits with failure if the command fails.

#### ok
Returns boolean true if the last `exec` was successful.

#### date
Returns the current date and time, formatted according to its argument.
The format is passed directly to the `go` date formatter:

```
{{date "2006-01-02"}}
```

Remember the [magic date string](https://golang.org/pkg/time/#Time.Format): `Mon Jan 2 15:04:05 -0700 MST 2006`.

#### split
Returns a slice of strings by splitting its argument by a given string, or space if not specified.

```
{{split "1,2,3"  ","}}
```

#### join
Returns a string by joining its slice argument with a given string, or space if not specified.


#### glob
Returns a list of files matching a given pattern, using [glob](https://golang.org/pkg/path/filepath/#Glob).
Accepts several glob patterns.
```
ASSETS={{range glob "assets/*"}}{{.}} {{end}}
```

#### replace
Expects three arguments, a pattern, a replacement and an operand. The operand can be either a single string or a list of strings.
Replace runs regex replacement using [Replace](https://golang.org/pkg/regexp/#Regexp.ReplaceAllString).

## Syntax highlighting

To make it more fun to edit `bygg` files in VS Code, I put together a basic syntax highlighting package. Search for "bygg syntax highlighting" in the marketplace.

You're welcome!

## C/C++ dependency check example

The following template uses the `g++` / `clang` -flags `-MM` and `-MT` to insert auto-generated dependencies:

```
{{-
/**
    "depbuild"
    Dependency build template, expects a string of comma-separated args:
    <module>, <source pattern>
**/
-}}
{{define "depbuild"}}

    {{$args := split . ","}}
    {{$module := index $args 0}}
    {{$sourcePattern := index $args 1}}
    {{$extraBuildFlags := index $args 2}}

    {{$env := (env)}}

    {{range glob $sourcePattern}}
        {{$src := .}}
        {{$pre := printf "%s/%s" $env.BUILD (replace "/" "_" $src)}}
        {{$obj := (replace ".c(pp)?$" ".o" $pre)}}
        {{$module}}_objects+={{$obj}}
        # Dependency magic
        {{mustexec $env.CPP (split $env.CPPFLAGS) "-MM" "-MT" $obj $src}}
        {{$obj}} <- {{$env.CPP}} {{$env.CPPFLAGS}} -c -o {{$obj}} {{$src}}
    {{end}}

{{end}}
{{/* depbuild */}}
```

In this example, the environment variables `CPP` and `CPPFLAGS` have been set up before invoking the template:
```
{{template "depbuild" "myproj,myproj/*.c"}}

myproj: $myproj_objects
```

## But why yet another build tool:question:

This started as a way to get portable builds for [Letarette](https://letarette.io), but it should work well for other `go` projects with similar needs.

Letarette is a `go` project, but it relies heavily on `sqlite3` and the `Snowball` library, both C libraries.

I started out using `make` and `bash`, which worked fine for a while.
It was a couple of iterations before the Linux and Mac builds worked the same, but when I got to Windows, I hit a wall.

So, I started thinking - could I script the build process in `go` instead?

As usual, I let it grow a little bit too far. But - it's still well below 1000 lines of code, and has no external dependencies.
