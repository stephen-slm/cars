# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [content/consumer/v1/consumer.proto](#content_consumer_v1_consumer-proto)
    - [CompileQueueResponse](#content-consumer-v1-CompileQueueResponse)
    - [CompileRequest](#content-consumer-v1-CompileRequest)
    - [CompileResultRequest](#content-consumer-v1-CompileResultRequest)
    - [CompileResultResponse](#content-consumer-v1-CompileResultResponse)
    - [GetSupportedLanguagesResponse](#content-consumer-v1-GetSupportedLanguagesResponse)
    - [GetTemplateRequest](#content-consumer-v1-GetTemplateRequest)
    - [GetTemplateResponse](#content-consumer-v1-GetTemplateResponse)
    - [PingResponse](#content-consumer-v1-PingResponse)
    - [SupportedLanguage](#content-consumer-v1-SupportedLanguage)
  
    - [ConsumerService](#content-consumer-v1-ConsumerService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="content_consumer_v1_consumer-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## content/consumer/v1/consumer.proto



<a name="content-consumer-v1-CompileQueueResponse"></a>

### CompileQueueResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="content-consumer-v1-CompileRequest"></a>

### CompileRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| language | [string](#string) |  |  |
| source | [string](#string) |  |  |
| standard_in_data | [string](#string) | repeated | This array of strings will be written to the standard input of the code when executing. Each array item is a line which will be written one after another. |
| expected_standard_out_data | [string](#string) | repeated | This is an array of expected output data, including data here that will result in a validation check on completion. If no items are added to the array then the status endpoint will return NoTest for the test status. Otherwise, a value related to the test result. |






<a name="content-consumer-v1-CompileResultRequest"></a>

### CompileResultRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="content-consumer-v1-CompileResultResponse"></a>

### CompileResultResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| language | [string](#string) |  |  |
| status | [string](#string) |  |  |
| test_status | [string](#string) |  |  |
| compile_ms | [int64](#int64) |  |  |
| runtime_ms | [int64](#int64) |  |  |
| runtime_memory_mb | [double](#double) |  |  |
| output | [string](#string) |  |  |
| output_error | [string](#string) |  |  |
| compiler_output | [string](#string) |  |  |






<a name="content-consumer-v1-GetSupportedLanguagesResponse"></a>

### GetSupportedLanguagesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| languages | [SupportedLanguage](#content-consumer-v1-SupportedLanguage) | repeated |  |






<a name="content-consumer-v1-GetTemplateRequest"></a>

### GetTemplateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| language | [string](#string) |  |  |






<a name="content-consumer-v1-GetTemplateResponse"></a>

### GetTemplateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| template | [string](#string) |  |  |






<a name="content-consumer-v1-PingResponse"></a>

### PingResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message | [string](#string) |  |  |






<a name="content-consumer-v1-SupportedLanguage"></a>

### SupportedLanguage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| language_code | [string](#string) |  | The language code send during the compile request, this is not the same as the display name. This is also the code used to get the template. |
| display_name | [string](#string) |  | The display name the user can be shown and will understand for example the display name could be C# and the code would be csharp. |





 

 

 


<a name="content-consumer-v1-ConsumerService"></a>

### ConsumerService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Ping | [.google.protobuf.Empty](#google-protobuf-Empty) | [PingResponse](#content-consumer-v1-PingResponse) |  |
| GetTemplate | [GetTemplateRequest](#content-consumer-v1-GetTemplateRequest) | [GetTemplateResponse](#content-consumer-v1-GetTemplateResponse) | GetTemplate is designed to allow consumers of the platform to serve the user with a template they can start from. This is more important for languages that require selective formatting or a main function. An example of these languages would be C&#43;&#43;, and C. |
| GetSupportedLanguages | [.google.protobuf.Empty](#google-protobuf-Empty) | [GetSupportedLanguagesResponse](#content-consumer-v1-GetSupportedLanguagesResponse) | GetSupportedLanguages will return a list of languages that can be exposed to the user. This response contains a display name for the language that will contain compiler information if important and will also return the code. The code is the value sent to the server when requesting to compile and run. |
| CreateCompileRequest | [CompileRequest](#content-consumer-v1-CompileRequest) | [CompileQueueResponse](#content-consumer-v1-CompileQueueResponse) | CompileQueueRequest is the core compile request endpoint. Calling into this will trigger the flow to run the user-submitted code. |
| GetCompileResultRequest | [CompileResultRequest](#content-consumer-v1-CompileResultRequest) | [CompileResultResponse](#content-consumer-v1-CompileResultResponse) | GetCompileResultRequest is required to be called after requesting to compile, all details about the running state and the final output of the compiling and execution are from this. |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

