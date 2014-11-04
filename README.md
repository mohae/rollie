Rollie
========

## Under development
Rollie is a Go package for Mustache templates.

## License
Rollie is provided under the BSD3 license. All right are reserved by The Rollie Authors. Any code written by The Go Authors retain their original copyright; no additional claims or warranties, explicit or implied, are made by The Rollie Authors. Please view the included LICENSE file for more information on the licensing itself.

As noted above, there is considerable code by The Go Authors in Rollie, as Rollie is based on Go's implementation of text templating with its `text/template` and `html/template` packages and sub-packages. Changes have been made to the lexing, parsing, and rendering process, and supporting methods and functions to implement Mustache templating.

## Usage

    go get github.com/mohae/rollie

    import (
        "github.com/mohae/rollie"
    )

    Parse
 
    ParseFile

    Render

    RenderFile



## Example implementation
[Mustax](https://github.com/mohae/mustax) is a CLI application for lexing, parsing, and rendering mustache templates. It serves both as a tool and a test harness for the [Go Rollie Mustache template package](https://github.com/mohae/rollie).

## Notes:
This package is new and not widely used, implemented. Please report any issues. You would be your typical awesome self if you could make a pull request that fixes said issue! Ya, you know your awesome and your awesomness makes you want to contribute.

## Next Version
### 0.1
This will be the initial version of the package. Rollie will pass all mustache spec tests-excluding the optional tests. This means no lambda support at 0.1.0. This represents a minimal implementation of the Mustache spec. Any additional functionality will be part of a later release.

