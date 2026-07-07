document.addEventListener('DOMContentLoaded', () => {
  // DOM Elements
  const jsonInput = document.getElementById('json-input');
  const inputLineNumbers = document.getElementById('input-line-numbers');
  const themeSelector = document.getElementById('theme-selector');
  const indentSelector = document.getElementById('indent-selector');
  const minifyCheckbox = document.getElementById('minify-checkbox');
  const btnFormat = document.getElementById('btn-format');
  const btnClear = document.getElementById('btn-clear');
  const btnCopy = document.getElementById('btn-copy');
  const btnDownload = document.getElementById('btn-download');
  const fileUpload = document.getElementById('file-upload');
  const dragOverlay = document.getElementById('drag-overlay');
  
  // Output and Error
  const outputViewer = document.getElementById('output-viewer');
  const errorAlert = document.getElementById('error-alert');
  const errorMessage = document.getElementById('error-message');
  const errorLocation = document.getElementById('error-location');
  const toast = document.getElementById('toast');

  // Stats Elements
  const statSize = document.getElementById('stat-size');
  const statDepth = document.getElementById('stat-depth');
  const statKeys = document.getElementById('stat-keys');
  const statObjects = document.getElementById('stat-objects');

  let formattedResult = ''; // Holds the raw formatted JSON

  // ───────────────────────────────────────────────────────────────────────────
  // 1. Theme Configuration
  // ───────────────────────────────────────────────────────────────────────────
  const savedTheme = localStorage.getItem('json-formatter-theme') || 'theme-neon-night';
  document.body.className = savedTheme;
  themeSelector.value = savedTheme;

  themeSelector.addEventListener('change', (e) => {
    const theme = e.target.value;
    document.body.className = theme;
    localStorage.setItem('json-formatter-theme', theme);
  });

  // ───────────────────────────────────────────────────────────────────────────
  // 2. Line Numbers Sync
  // ───────────────────────────────────────────────────────────────────────────
  function updateLineNumbers() {
    const lines = jsonInput.value.split('\n').length;
    let lineNumbersHTML = '';
    for (let i = 1; i <= lines; i++) {
      lineNumbersHTML += i + '<br>';
    }
    inputLineNumbers.innerHTML = lineNumbersHTML;
  }

  jsonInput.addEventListener('input', updateLineNumbers);
  jsonInput.addEventListener('scroll', () => {
    inputLineNumbers.scrollTop = jsonInput.scrollTop;
  });

  // Initialize line numbers
  updateLineNumbers();

  // ───────────────────────────────────────────────────────────────────────────
  // 3. Drag and Drop File Upload
  // ───────────────────────────────────────────────────────────────────────────
  // Prevent default drag behaviors
  ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
    document.addEventListener(eventName, preventDefaults, false);
  });

  function preventDefaults(e) {
    e.preventDefault();
    e.stopPropagation();
  }

  // Highlight drop area when item is dragged over
  jsonInput.addEventListener('dragenter', () => dragOverlay.classList.add('active'), false);
  jsonInput.addEventListener('dragover', () => dragOverlay.classList.add('active'), false);
  dragOverlay.addEventListener('dragleave', () => dragOverlay.classList.remove('active'), false);

  // Handle dropped files
  dragOverlay.addEventListener('drop', (e) => {
    dragOverlay.classList.remove('active');
    const dt = e.dataTransfer;
    const files = dt.files;
    if (files.length > 0) {
      handleFile(files[0]);
    }
  }, false);

  // Handle traditional file input upload
  fileUpload.addEventListener('change', (e) => {
    if (e.target.files.length > 0) {
      handleFile(e.target.files[0]);
    }
  });

  function handleFile(file) {
    const reader = new FileReader();
    reader.onload = (e) => {
      jsonInput.value = e.target.result;
      updateLineNumbers();
      // Auto-trigger format after upload
      formatJSON();
    };
    reader.onerror = () => {
      showError('Failed to read upload file.');
    };
    reader.readAsText(file);
  }

  // ───────────────────────────────────────────────────────────────────────────
  // 4. Formatting and Syntax Highlighting API
  // ───────────────────────────────────────────────────────────────────────────
  async function formatJSON() {
    const rawVal = jsonInput.value.trim();
    if (!rawVal) {
      showEmptyState();
      return;
    }

    try {
      const indent = parseInt(indentSelector.value, 10);
      const minify = minifyCheckbox.checked;

      const response = await fetch('/api/format', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          jsonData: rawVal,
          indent: indent,
          minify: minify
        })
      });

      const data = await response.json();

      if (!data.success) {
        showError(data.errorMessage, data.errorLine, data.errorCol);
        return;
      }

      // Success
      formattedResult = data.formatted;
      hideError();
      renderFormattedOutput(formattedResult);
      renderStats(data.stats);

    } catch (err) {
      showError('Communication error: failed to reach formatting server.');
      console.error(err);
    }
  }

  btnFormat.addEventListener('click', formatJSON);

  // Render stats
  function renderStats(stats) {
    if (!stats) return;
    
    // Format file size
    let sizeStr = '';
    if (stats.size < 1024) {
      sizeStr = stats.size + ' B';
    } else if (stats.size < 1024 * 1024) {
      sizeStr = (stats.size / 1024).toFixed(2) + ' KB';
    } else {
      sizeStr = (stats.size / (1024 * 1024)).toFixed(2) + ' MB';
    }

    statSize.textContent = sizeStr;
    statDepth.textContent = stats.maxDepth;
    statKeys.textContent = stats.keyCount;
    statObjects.textContent = stats.objectCount;
  }

  // Hide empty state and render output
  function renderFormattedOutput(jsonStr) {
    const highlighted = highlightJSON(jsonStr);
    outputViewer.innerHTML = `<code class="mono">${highlighted}</code>`;
  }

  function highlightJSON(jsonStr) {
    // Escape HTML first
    let html = jsonStr
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
    
    const regex = /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+-]?\d+)?)/g;
    
    return html.replace(regex, function (match) {
      let cls = 'json-number';
      if (/^"/.test(match)) {
        if (/:$/.test(match)) {
          cls = 'json-key';
        } else {
          cls = 'json-string';
        }
      } else if (/true|false/.test(match)) {
        cls = 'json-boolean';
      } else if (/null/.test(match)) {
        cls = 'json-null';
      }
      return '<span class="' + cls + '">' + match + '</span>';
    });
  }

  // ───────────────────────────────────────────────────────────────────────────
  // 5. Error & States Management
  // ───────────────────────────────────────────────────────────────────────────
  function showError(msg, line = 0, col = 0) {
    errorAlert.classList.remove('hidden');
    errorMessage.textContent = msg;
    if (line > 0 && col > 0) {
      errorLocation.textContent = `Line: ${line}, Col: ${col}`;
      errorLocation.classList.remove('hidden');
    } else {
      errorLocation.classList.add('hidden');
    }
    outputViewer.innerHTML = `<div class="empty-state"><p class="text-secondary">Validation failed. View syntax error above.</p></div>`;
    clearStats();
  }

  function hideError() {
    errorAlert.classList.add('hidden');
  }

  function showEmptyState() {
    hideError();
    clearStats();
    outputViewer.innerHTML = `
      <div class="empty-state">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>
        <p>Formatted JSON and structural tree visualization will render here.</p>
      </div>`;
    formattedResult = '';
  }

  function clearStats() {
    statSize.textContent = '-';
    statDepth.textContent = '-';
    statKeys.textContent = '-';
    statObjects.textContent = '-';
  }

  // ───────────────────────────────────────────────────────────────────────────
  // 6. Action Buttons Handlers
  // ───────────────────────────────────────────────────────────────────────────
  // Clear button
  btnClear.addEventListener('click', () => {
    jsonInput.value = '';
    updateLineNumbers();
    showEmptyState();
  });

  // Copy button
  btnCopy.addEventListener('click', () => {
    if (!formattedResult) {
      showToast('No formatted JSON to copy');
      return;
    }

    navigator.clipboard.writeText(formattedResult).then(() => {
      showToast('Copied to clipboard!');
    }).catch(err => {
      console.error('Copy failed:', err);
      // Fallback
      const textArea = document.createElement('textarea');
      textArea.value = formattedResult;
      document.body.appendChild(textArea);
      textArea.select();
      try {
        document.execCommand('copy');
        showToast('Copied to clipboard!');
      } catch (err2) {
        showToast('Failed to copy.');
      }
      document.body.removeChild(textArea);
    });
  });

  // Download button
  btnDownload.addEventListener('click', () => {
    if (!formattedResult) {
      showToast('No formatted JSON to download');
      return;
    }

    const blob = new Blob([formattedResult], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'formatted.json';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    showToast('Download started');
  });

  function showToast(msg) {
    toast.textContent = msg;
    toast.classList.remove('hidden');
    setTimeout(() => {
      toast.classList.add('hidden');
    }, 2000);
  }
});
