# `/internal`

Historically Go programs (or libraries) could be imported by any other program
(or library) providing that the imported symbol is
[exported](https://www.ardanlabs.com/blog/2014/03/exportedunexported-identifiers-in-go.html).

In order to allow packages to export some functionalities to other packages of
the same program (or library), but to prevent other programs (or libraries) from
importing them directly, a
[proposal](https://docs.google.com/document/d/1e8kOo3r51b2BWtTs_1uADIA5djfXhPT36s6eHVRIvaU/edit#!)
to define an `/internal` directory came up: the Go compiler will not allow
importing anything that lives in this directory from any project external to the
one that owns it.

This is the code you don't want others importing in their applications or
libraries. See the Go 1.4 [`release
notes`](https://golang.org/doc/go1.4#internalpackages) for more details. Note
that you are not limited to the top level `internal` directory. You can have
more than one `internal` directory at any level of your project tree.

How you want to structure your project in this directory is up to you, but this
is probably where most of the Go code will be.

Examples:

*
