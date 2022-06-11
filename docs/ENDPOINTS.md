Below includes the documentation of the existing API endpoints which are exposed from the project.

- [Compile](#compile)
    * [Compile Request](#compile-request)
        + [Request](#request)
        + [Response](#response)
        + [stdin data](#stdin-data)
        + [expected stdout data](#expected-stdout-data)
    * [Compile Response](#compile-response)
        + [Response](#response-1)
        + [Possible Status Values:](#possible-status-values-)
        + [Possible Test Status Values:](#possible-test-status-values-)
- [Templates](#templates)
    * [Language Template](#language-template)


# Compile

## Compile Request

This is the core compile request endpoint. Calling into this will trigger the flow to run the user-submitted code.

URL: `POST - /compile/`  
Response: Get the request's id, and use this to gather updates on the execution.

Example: `POST - /compile/`

### Request

```JSON
{
  "language": "string",
  "source_code": "string",
  "stdin_data": [
	"string"
  ],
  "expected_stdout_data": [
	"string"
  ]
}
```

### Response

```JSON
{
  "id": "7197011b-7744-486c-b3e2-e86fd42d0a62"
}
```

### stdin data

This array of strings will be written to the standard input of the code when executing. Each array item is a line which
will be written one after another.

### expected stdout data

This is an array of expected output data, including data here that will result in a validation check on completion. If
no items are added to the array then the status endpoint will return `NoTest` for the test status. Otherwise, a value
related to the test result.

## Compile Response

This endpoint is required to be called after requesting to compile, all details about the running state and the final
output of the compiling and execution are from this.

URL: `GET - /compile/{id}`  
Response: Get the state of a compile request based on the compile request-id.

Example: `GET - /compile/56ea6176-d23f-4561-b937-b16c6a8434ef`

### Response

```json
{
  "status": "string",
  "test_status": "string",
  "compile_ms": 0,
  "runtime_ms": 0,
  "language": "string",
  "output": "string"
}
```

### Possible Status Values:

* NotRan
* Created
* Running
* Killing
* Killed
* Finished
* MemoryConstraintExceeded
* TimeLimitExceeded
* ProvidedTestFailed
* CompilationFailed
* RunTimeError
* NonDeterministicError

### Possible Test Status Values:

* NoTest
* TestNotRan
* TestFailed
* TestPassed

# Templates

API endpoints to support the gathering and working with language templates.

## Language Template

This endpoint is designed to allow consumers of the platform to serve the user with a template they can start from. This
is more important for languages that require selective formatting or a `main` function. An example of these languages
would be Haskell, C++, and C.

URL: `GET - /templates/{language}`  
Response: A usable template for that language that will run when attempting to compile

Example: `GET - /templates/cpp`

```cpp
#include <iostream>

int main() {
    std::cout << "Hello World!";
    return 0;
}
```
