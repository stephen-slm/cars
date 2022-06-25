const state = {
    editor: ace.edit('editor'),

    languageAceMapping: {
        "python2": "python",
        "c": "c_cpp",
        "cpp": "c_cpp",
        "node": "javascript",
        "fsharp": "fsharp",
    },

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

function isFinishingStatus(status) {
    const endingStatusValues =["Finished", "Killed", "MemoryConstraintExceeded", "TimeLimitExceeded",
        "CompilationFailed", "RunTimeError", "NonDeterministicError"]

    return endingStatusValues.includes(status)
}

function disableOrEnableRunButton(value) {
    document.getElementById("run").disabled = value
}

function clearAllOutput() {
    state.outputDiv.innerText = ''
    state.compilerOutputDiv.innerText = ''
    state.codeStatus.innerText = ''
    state.codeTestStatus.innerText = ''
}

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


async function performExecutionLookup(id, executionCount = 0) {
    if (executionCount > 20) {
        return handleCodeExecutionTakenTooLong(id, executionCount)
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

async function handleLanguageSelectionChange(event) {
    return await updateEditorForLanguage(event.target.value.toLowerCase(), false)
}


async function handleResetCodeTemplatePressed() {
    return await updateEditorForLanguage(state.currentLanguage, true)
}

function handleCodeExecutionTakenTooLong(id, executionCount) {

}


async function handleCompileRequest(event) {
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

async function updateEditorForLanguage(language, force) {
    state.currentLanguage = language;

    const mappingValue = state.languageAceMapping[language] || language

    const template = await getLanguageTemplateValue(language)

    state.editor.getSession().setMode(`ace/mode/${mappingValue}`);
    state.editor.getSession().setValue(template)
}

function configureEditor() {
    state.editor.setTheme('ace/theme/dracula');
    state.editor.session.setUseWrapMode(true);

    state.editor.setFontSize(16)

    state.editor.renderer.setOptions({
        showPrintMargin: true,
    });

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

async function getExecutionCurrentStatus(id) {
    const response = await fetch(`http://localhost:8080/compile/${id}`)
    return await response.json()
}

async function getLanguageTemplateValue(language) {
    const response = await fetch(`http://localhost:8080/languages/${language}/template`)
    return await response.text()
}


async function getCurrentLanguageSupport() {
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

    await getCurrentLanguageSupport()
    await updateEditorForLanguage(state.languages[0]["language_code"])

    // setup event handlers
    state.runButton.addEventListener("click", handleCompileRequest)
    state.resetCodeButton.addEventListener("click", handleResetCodeTemplatePressed)
    state.enableTestCheck.addEventListener("click", handleOnTestEnableClick)
    state.languageModes.addEventListener("change", handleLanguageSelectionChange);
}


window.addEventListener("load", init)
