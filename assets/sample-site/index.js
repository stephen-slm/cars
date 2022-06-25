const state = {
    // Third party editor to help show how CARS works.
    //
    // https://ace.c9.io/
    editor: ace.edit('editor'),

    // Helper for the Ace editor, this a third party lib to help show how CARS
    // works. This is a mapping between types to allow syntax highlighting to
    // work as expected.
    languageAceMapping: {
        "python2": "python",
        "c": "c_cpp",
        "cpp": "c_cpp",
        "node": "javascript",
        "fsharp": "fsharp",
    },

    /**
     * Bunch of helpers to bind elements to simplify code
     */

    outputDiv: document.getElementById("code-output"),
    compilerOutputDiv: document.getElementById("compiler-output"),

    languageModes: document.getElementById("language-modes"),
    codeRuntimeMs: document.getElementById("code-runtime"),
    compilerRuntimeMs: document.getElementById("compiler-runtime"),
    codeStatus: document.getElementById("execution-status"),
    codeTestStatus: document.getElementById("execution-test-status"),
    enableTestCheck: document.getElementById("enable-test"),
    runButton: document.getElementById("run"),
    resetCodeButton: document.getElementById("reset-code"),


    testStandardInput: document.getElementById("standard-input"),
    testExpectedOutput: document.getElementById("expected-output"),
}


/////////////////////////////////////////////////////////////
// HELPERS
/////////////////////////////////////////////////////////////

/**
 * More than one value can be marked as "finished" for a run. This just helps
 * us handle all finishing cases. The simple site does not do anything special
 * based on output.
 *
 * @param status
 * @returns {boolean}
 */
function isFinishingStatus(status) {
    const endingStatusValues = ["Finished", "Killed", "MemoryConstraintExceeded", "TimeLimitExceeded",
        "CompilationFailed", "RunTimeError", "NonDeterministicError", "Failed"]

    return endingStatusValues.includes(status)
}

/**
 * Helper to enable or disable the run button which is done during and
 * after runs to stop the change of duplicate runs. (this is supported, but
 * it would make the site more complicated than needed to show it off).
 * @param value
 */
function disableOrEnableRunButton(value) {
    document.getElementById("run").disabled = value
}

/**
 * Helper to clear all fields before running
 */
function clearAllOutput() {
    state.outputDiv.innerText = ''
    state.compilerOutputDiv.innerText = ''
    state.codeStatus.innerText = ''
    state.codeTestStatus.innerText = ''
}

/**
 * Helper to write out the result of a run to all the correct locations.
 * @param output
 */
function writeCompileStatusOutputToDisplay(output) {
    console.log(JSON.stringify(output, null, 2))

    let outputContent = output["output"] || ""

    if (!isFinishingStatus(output.status)) {
        outputContent = `${state.outputDiv.innerText}${output.status}...\n`
    }

    state.compilerRuntimeMs.innerText = output["compile_ms"] || "0"
    state.codeRuntimeMs.innerText = output["runtime_ms"] || "0"
    state.codeStatus.innerText = output.status
    state.codeTestStatus.innerText = output["test_status"]

    state.outputDiv.innerText = outputContent
    state.compilerOutputDiv.innerText = output["compiler_output"] || ""
}


/**
 * Perform the fetch execution loop to gain details about the run and compile
 * process, ensuring to update the user during each 500ms tick.
 *
 * This is called recursively via `setTimeout`.
 *
 * @param id
 * @param executionCount
 * @returns {Promise<void>}
 */
async function performExecutionLookup(id, executionCount = 0) {
    if (executionCount > 20) {
        return handleCodeExecutionTakenTooLong(id)
    }

    const output = await getExecutionCurrentStatus(id)
    writeCompileStatusOutputToDisplay(output)

    if (isFinishingStatus(output.status)) {
        return disableOrEnableRunButton(false)
    }

    setTimeout(performExecutionLookup.bind(this, id, executionCount + 1), 500)
}


/////////////////////////////////////////////////////////////
// EVENT CALLS
/////////////////////////////////////////////////////////////

/**
 * Handle a selection change in the languages' dropdown. Switching the code and
 * language (trying its best to use any caching and falling back to the template)
 *
 * @param event
 * @returns {Promise<void>}
 */
async function handleLanguageSelectionChange(event) {
    return await updateEditorForLanguage(event.target.value.toLowerCase(), false)
}

/**
 * Handle on code change events which will update our cached code. Helps anyone
 * to use the sample site to switch between languages without loosing progress.
 */
function handleEditorValueChange() {
    const value = state.editor.getValue()
   localStorage.setItem(state.currentLanguage, value)
}

/**
 * Handle a "Reset Code" click which will force the current code to be set
 * to the template while also clearing any cached code.
 *
 * @returns {Promise<void>}
 */
async function handleResetCodeTemplatePressed() {
    return await updateEditorForLanguage(state.currentLanguage, true)
}

/**
 * Handle when the server faulted for any reason and it's not updating the
 * status. This is triggered when we have gone way past our duration.
 *
 * TODO: update this to only count the 10 seconds AFTER the code has actually
 *  started since it could just be in the queue for running since the queue
 *  size is fairly small at the moment.
 * @param id
 */
function handleCodeExecutionTakenTooLong(id) {
    disableOrEnableRunButton(false)


    writeCompileStatusOutputToDisplay({
        output: `${id} - execution has taken too long and we stopped requesting details,
        please try again (something probably faulted).`,
        status: "Failed",
        "test_status": "Failed"
    })
}


/**
 * Triggered by clicking "Run", this will take all the code, language and
 *  test information if enabled and trigger a compile request.
 *
 *  Once complete this will trigger a feedback loop gathering details about
 *  the execution of the run and display this back to the user during and
 *  after completion.
 * @returns {Promise<void>}
 */
async function handleCompileRequest() {
    disableOrEnableRunButton(true)
    clearAllOutput()

    const code = state.editor.getValue();
    const language = state.currentLanguage

    const stdInData = state.enableTestCheck.checked
        ? state.testStandardInput.value.split('\n')
        : []

    const expectedOut = state.enableTestCheck.checked
        ? state.testExpectedOutput.value.split('\n')
        : []

    const id = await compileEditorCode(language, code, stdInData, expectedOut)
    console.log(`language: ${language} - id: ${id}`)

    setTimeout(performExecutionLookup.bind(this, id, 1), 500)
}

function handleOnTestEnableClick() {
    state.testStandardInput.disabled = !state.enableTestCheck.checked
    state.testExpectedOutput.disabled = !state.enableTestCheck.checked
}


/////////////////////////////////////////////////////////////
// EDITOR ACTIONS
/////////////////////////////////////////////////////////////

/**
 * Update the editor to the new language, this will pull any cached code related
 * to that language if set otherwise fallback to the template. This can be force
 * to the template with "force" as true.
 *
 * @param language
 * @param force
 * @returns {Promise<void>}
 */
async function updateEditorForLanguage(language, force) {
    localStorage.setItem("currentLanguage", language)
    state.currentLanguage = language;

    const mappingValue = state.languageAceMapping[language] || language
   let  userLastEditedCode = localStorage.getItem(language)

    if (userLastEditedCode == null || userLastEditedCode.trim() === "" || force) {
         userLastEditedCode = await getLanguageTemplateValue(language)
    }

    state.editor.getSession().setMode(`ace/mode/${mappingValue}`);
    state.editor.getSession().setValue(userLastEditedCode)
}

/**
 * Set up the editor to the default theme, font, margin and event handlers.
 */
function configureEditor() {
    state.editor.setTheme('ace/theme/dracula');
    state.editor.session.setUseWrapMode(true);

    state.editor.setFontSize(16)

    state.editor.renderer.setOptions({
        showPrintMargin: true,
    });

    state.editor.session.on("change", handleEditorValueChange)

}


/////////////////////////////////////////////////////////////
// API CALLS
/////////////////////////////////////////////////////////////

/**
 * Execute a compile request to the backend, returning an id value used to gather
 * updates regarding  the process.
 *
 * @param language
 * @param code
 * @param stdIn
 * @param expectedStdOut
 * @returns {Promise<string>} The id of the execution
 */
async function compileEditorCode(language, code, stdIn = [], expectedStdOut = []) {
    const body = {
        language,
        "source_code": code,
        "stdin_data": stdIn,
        "expected_stdout_data": expectedStdOut
    }

    console.log(JSON.stringify(body, null, 2))

    const response = await fetch('http://localhost:8080/compile', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(body)
    })

    const {id} = await response.json()
    return id
}

/**
 * Get the current execution status of a given compile.
 * @param id The id which was returned by the compile request.
 * @returns {Promise}
 */
async function getExecutionCurrentStatus(id) {
    const response = await fetch(`http://localhost:8080/compile/${id}`)
    return await response.json()
}

/**
 * Get a supporting template to help the user get started for a given language.
 * @param language
 * @returns {Promise<string>}
 */
async function getLanguageTemplateValue(language) {
    const response = await fetch(`http://localhost:8080/languages/${language}/template`)
    return await response.text()
}


/**
 * Get the list of all supporting languages and the related display name. This can be used
 * to easily set up any UI to display what is supported.
 *
 * This will set up the modes' element to include all the values.
 *
 * @returns {Promise<void>}
 */
async function setupCurrentLanguageSupport() {
    const response = await fetch('http://localhost:8080/languages')
    const json = await response.json()

    state.languages = json
    const root = document.getElementById("language-modes")

    for (const language of json) {
        const newOption = document.createElement("option");
        newOption.value = language['language_code']
        newOption.innerText = language['display_name']

        root.appendChild(newOption)
    }
}

async function init() {
    configureEditor()

    await setupCurrentLanguageSupport()

   const storedLanguage = localStorage.getItem("currentLanguage")
   const language = storedLanguage != null ? storedLanguage : state.languages[0]["language_code"]

    // force the modes to display the correct language
    state.languageModes.value = language

    await updateEditorForLanguage(language, false)

    // setup event handlers
    state.runButton.addEventListener("click", handleCompileRequest)
    state.resetCodeButton.addEventListener("click", handleResetCodeTemplatePressed)
    state.enableTestCheck.addEventListener("click", handleOnTestEnableClick)
    state.languageModes.addEventListener("change", handleLanguageSelectionChange);
}


window.addEventListener("load", init)
