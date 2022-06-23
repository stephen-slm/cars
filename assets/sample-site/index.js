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

/////////////////////////////////////////////////////////////
// EDITOR ACTIONS
/////////////////////////////////////////////////////////////

async function updateEditorForLanguage(language) {
    const mappingValue = state.languageAceMapping[language] || language

    const template = await getLanguageTemplateValue(language)

    state.editor.getSession().setMode(`ace/mode/${mappingValue}`);
    state.editor.getSession().setValue(template)
}

function configureEditor() {
    state.editor.setTheme('ace/theme/dracula');
    state.editor.session.setUseWrapMode(true);

    state.editor.renderer.setOptions({
        showPrintMargin: true,
    });

    state.editor.session.on('change', handleEditorValueChange);

    document.querySelector('#language-modes').addEventListener("change", handleLanguageSelectionChange);
}


/////////////////////////////////////////////////////////////
// API CALLS
/////////////////////////////////////////////////////////////

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
}


window.addEventListener("load", init)
