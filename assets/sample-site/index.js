let editor = ace.edit('editor');
editor.setTheme('ace/theme/dracula');
editor.session.setMode('ace/mode/markdown');
editor.session.setUseWrapMode(true);
editor.container.style.lineHeight = 1.15;
editor.renderer.setOptions({
    showPrintMargin: false,
    fontSize: '.875rem',
    fontFamily: '"IBM Plex Mono", monospace'
});
editor.clearSelection();

editor.session.on('change', () => {
    document.querySelector("#preview").innerHTML = editor.getValue()
});

document.querySelector('#ace-mode').addEventListener("change", function (event) {
    editor.getSession().setMode(`ace/mode/${event.target.value.toLowerCase()}`);
});

