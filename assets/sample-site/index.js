const state = {
    editor: ace.edit('editor'),

    languageAceMapping: {
        "python2": "python",
        "c": "c_cpp",
        "cpp": "c_cpp",
        "node": "javascript",
        "fsharp": "fsharp",
    }
}

/////////////////////////////////////////////////////////////
// EVENT CALLS
/////////////////////////////////////////////////////////////

async function handleLanguageSelectionChange(event) {
    return await updateEditorForLanguage(event.target.value.toLowerCase())
}

function handleEditorValueChange() {
    document.querySelector("#compiler-output").innerHTML = state.editor.getValue()
}

async function handleCompileRequest(event) {
    event.target.disabled = true

    const code = state.editor.getValue();
    const language = state.currentLanguage

    const id = await compileEditorCode(language, code, [], [])
    console.log(`language: ${language} - id: ${id}`)

    event.target.disabled = false
}


/////////////////////////////////////////////////////////////
// EDITOR ACTIONS
/////////////////////////////////////////////////////////////

async function updateEditorForLanguage(language) {
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

    state.editor.session.on('change', handleEditorValueChange);

    document.querySelector('#language-modes').addEventListener("change", handleLanguageSelectionChange);
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
    const response = await fetch('http://localhost:8080/compile', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
            language,
            "source_code": code,
            "stdin_data": stdIn,
            "expected_stdout_data": expectedStdOut
        })
    })

    const {id} = await response.json()
    return id
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
    document.getElementById("run").addEventListener("click", handleCompileRequest)
}


window.addEventListener("load", init)
