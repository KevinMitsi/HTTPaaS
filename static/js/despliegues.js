let hostAccepted = false;
let currentHost = '';
let selectedFile = null;

function checkHost() {
  const value = document.getElementById('hostInput').value.trim();
  document.getElementById('domainPreview').textContent = value ? `${value}.cloud.local` : '‹hostname›.cloud.local';
  document.getElementById('btnAceptar').disabled = value.length < 2;
}

function aceptarHost() {
  const value = document.getElementById('hostInput').value.trim();
  if (!value) {
    return;
  }

  currentHost = value;
  hostAccepted = true;

  document.getElementById('card1').classList.remove('active');
  document.getElementById('card1').classList.add('done');
  document.getElementById('badge1').classList.replace('active', 'done');
  document.getElementById('badge1').textContent = '✓';
  document.getElementById('status1').style.display = 'block';
  document.getElementById('body1').classList.add('collapsed');
  document.getElementById('hostInput').readOnly = true;

  const card2 = document.getElementById('card2');
  card2.style.opacity = '1';
  card2.style.pointerEvents = 'auto';
  card2.classList.add('active');
  document.getElementById('badge2').classList.replace('pending', 'active');
  document.getElementById('body2').classList.remove('collapsed');

  showToast(`Host "${value}.cloud.local" configurado`, 'success');
  renderRightPanel();
}

function resetStep1() {
  hostAccepted = false;
  currentHost = '';
  selectedFile = null;

  document.getElementById('card1').classList.add('active');
  document.getElementById('card1').classList.remove('done');
  document.getElementById('badge1').classList.replace('done', 'active');
  document.getElementById('badge1').textContent = '1';
  document.getElementById('status1').style.display = 'none';
  document.getElementById('body1').classList.remove('collapsed');
  document.getElementById('hostInput').readOnly = false;
  document.getElementById('hostInput').value = '';

  document.getElementById('card2').style.opacity = '.5';
  document.getElementById('card2').style.pointerEvents = 'none';
  document.getElementById('card2').classList.remove('active');
  document.getElementById('badge2').classList.replace('active', 'pending');
  document.getElementById('body2').classList.add('collapsed');
  removeFile();
  checkHost();
}

function handleFile(input) {
  const file = input.files[0];
  if (!file) {
    return;
  }
  if (!file.name.endsWith('.zip') && !file.name.endsWith('.tar.gz')) {
    showToast('Solo se aceptan archivos .zip o .tar.gz', 'error');
    return;
  }

  selectedFile = file;
  const sizeMB = (file.size / 1048576).toFixed(1);
  document.getElementById('dropArea').style.display = 'none';
  document.getElementById('fileSelected').classList.add('show');
  document.getElementById('fscName').textContent = file.name;
  document.getElementById('fscSize').textContent = `${sizeMB} MB · Preparado para subir`;
  document.getElementById('btnPublicar').disabled = false;
}

function removeFile() {
  selectedFile = null;
  document.getElementById('fileInput').value = '';
  document.getElementById('dropArea').style.display = 'block';
  document.getElementById('fileSelected').classList.remove('show');
  document.getElementById('btnPublicar').disabled = true;
}

const dropArea = document.getElementById('dropArea');
dropArea.addEventListener('dragover', (event) => {
  event.preventDefault();
  dropArea.style.borderColor = 'var(--accent)';
  dropArea.style.background = 'var(--accent-lt)';
});
dropArea.addEventListener('dragleave', () => {
  dropArea.style.borderColor = '';
  dropArea.style.background = '';
});
dropArea.addEventListener('drop', (event) => {
  event.preventDefault();
  dropArea.style.borderColor = '';
  dropArea.style.background = '';
  const file = event.dataTransfer.files[0];
  if (file) {
    handleFile({ files: [file] });
  }
});

async function publicar() {
  if (!currentHost || !selectedFile) {
    showToast('Debes completar el host y subir un archivo antes de publicar', 'error');
    return;
  }

  const btn = document.getElementById('btnPublicar');
  const log = document.getElementById('progressLog');
  btn.innerHTML = '<span class="spinner"></span> Publicando...';
  btn.disabled = true;
  log.style.display = 'block';
  log.innerHTML = '';
  appendProgress('→ Enviando solicitud al backend...');

  try {
    const result = await apiProvisionar(currentHost, selectedFile);
    appendProgress(`✓ Instancia creada: ${result.instance.domain}`);
    showToast(`✓ ${result.instance.domain} desplegado correctamente`, 'success');

    markStep2Done();
    await renderRightPanel();

    window.setTimeout(() => {
      if (window.confirm('¿Deseas crear otro despliegue?')) {
        window.location.reload();
      } else {
        window.location.href = 'instancias.html';
      }
    }, 1200);
  } catch (error) {
    appendProgress(`✗ ${error.message}`);
    showToast(error.message, 'error');
    btn.innerHTML = '<svg width="13" height="13" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM6.293 6.707a1 1 0 010-1.414l3-3a1 1 0 011.414 0l3 3a1 1 0 01-1.414 1.414L11 5.414V13a1 1 0 11-2 0V5.414L7.707 6.707a1 1 0 01-1.414 0z" clip-rule="evenodd"/></svg> Publicar';
    btn.disabled = false;
  }
}

function appendProgress(message) {
  const log = document.getElementById('progressLog');
  const line = document.createElement('div');
  line.className = 'log-line';
  line.textContent = message;
  log.appendChild(line);
  log.scrollTop = log.scrollHeight;
}

function markStep2Done() {
  document.getElementById('badge2').classList.replace('active', 'done');
  document.getElementById('badge2').textContent = '✓';
  document.getElementById('card2').classList.remove('active');
  document.getElementById('card2').classList.add('done');
  document.getElementById('status2').style.display = 'block';
}

function initFromQuery() {
  const params = new URLSearchParams(window.location.search);
  const host = params.get('host');
  if (host) {
    document.getElementById('hostInput').value = host;
    checkHost();
  }
}

async function renderRightPanel() {
  const panel = document.getElementById('rightPanel');
  try {
    const instances = await apiListInstancias();
    if (!Array.isArray(instances) || instances.length === 0) {
      panel.innerHTML = `
        <div class="panel-empty">
          <div class="panel-empty-icon">🖥️</div>
          <p>No hay instancias activas.<br>Las instancias aparecerán aquí una vez finalice la publicación.</p>
        </div>`;
      return;
    }

    panel.innerHTML = instances.map((instance) => `
      <div class="instance-item">
        <span class="online-dot"></span>
        <a href="${instance.domain}" target="_blank" rel="noreferrer" class="inst-domain">${instance.domain}</a>
        <div class="inst-meta">
          <span>${instance.ip}</span>
          <span>·</span>
          <span>${instance.host}</span>
        </div>
      </div>
    `).join('');
  } catch (error) {
    panel.innerHTML = `
      <div class="panel-empty">
        <div class="panel-empty-icon">⚠️</div>
        <p>${error.message}</p>
      </div>`;
  }
}

checkHost();
initFromQuery();
renderRightPanel();
