# `/pkg`

The `pkg` directory should only contain library code that's intended and suitable to be used by external applications 
(e.g. `/pkg/serviceclient`). By default, you should only put library code in `/internal` unless you are 100% sure that
the library will be imported by another service straight away. It is a lot easier to move a package out of `/internal`
and into `/pkg` than the other way around. By putting code in `/internal`, Go enforces that it can only be used by your
project and not imported by another project. 

In the past the `pkg` directory has been used as a home for libraries which aren't specific to the project but also aren't
really in a suitable state to import into another project. This creates a lot of fragmentation between the `pkg` and 
`internal` directories, and often makes the project more difficult to navigate which is why we're discouraging the use
of the `pkg` folder except in rare circumstances.

An example of a library that makes sense to keep in `pkg` is a client library for interacting with your service, since
that will always need to be imported by another service.

The `pkg` directory origins: The old Go source code used to use `pkg` for its packages and then various Go projects in 
the community started copying the pattern (see [`this`](https://twitter.com/bradfitz/status/1039512487538970624) Brad 
Fitzpatrick's tweet for more context).
