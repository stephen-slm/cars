# `/cmd`

The `cmd` directory contains the main applications (or entry points) for this project. Each directory inside `/cmd` should
be built into a separate executable. The directory name for each application should match the name of the executable you 
want to have (e.g. `/cmd/foo`).

You shouldn't put a lot of code into the application directory, just the code necessary to initialise and run your
application. It's a common pattern to have a small `main` function that imports and invokes the code from the `/internal`
directory and nothing else.
