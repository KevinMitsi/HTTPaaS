let hostAccepted = false;
let currentHost = '';
let selectedFile = null;
let instances = [];

function handleHostInput() {
  const value = document.getElementById('hostInput').value.trim();
  document.getElementById('btnAceptar').disabled = value.length < 2;
}

function aceptarHost() {
  const value = document.getElementById('hostInput').value.trim();
  if (!value) {
    return;
  }

  currentHost = value;
  hostAccepted = true;

  document.getElementById('step2Section').classList.remove('disabled');
  document.getElementById('hostInput').readOnly = true;
  document.getElementById('btnAceptar').textContent = '✓ Aceptado';
  document.getElementById('btnAceptar').style.background = '#16a34a';
  document.getElementById('btnAceptar').disabled = true;

  showToast(`Host "${value}" configurado. Ahora suba el contenido web.`, 'success');
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
  document.getElementById('fileName').textContent = file.name;
  document.getElementById('fileSize').textContent = `${sizeMB} MB · Preparado para subir`;
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
  btn.innerHTML = '<span class="spinner"></span> Publicando...';
  btn.disabled = true;

  try {
    const result = await apiProvisionar(currentHost, selectedFile);
    showToast(`✓ ${result.instance.domain} desplegado correctamente`, 'success');
    await loadInstances();
    resetForm();
  } catch (error) {
    showToast(error.message, 'error');
    btn.innerHTML = '<svg width="13" height="13" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM6.293 6.707a1 1 0 010-1.414l3-3a1 1 0 011.414 0l3 3a1 1 0 01-1.414 1.414L11 5.414V13a1 1 0 11-2 0V5.414L7.707 6.707a1 1 0 01-1.414 0z" clip-rule="evenodd"/></svg> Publicar';
    btn.disabled = false;
  }
}

function resetForm() {
  hostAccepted = false;
  currentHost = '';
  selectedFile = null;

  document.getElementById('hostInput').value = '';
  document.getElementById('hostInput').readOnly = false;
  document.getElementById('btnAceptar').textContent = 'Aceptar';
  document.getElementById('btnAceptar').style.background = '';
  document.getElementById('btnAceptar').disabled = true;
  document.getElementById('step2Section').classList.add('disabled');
  removeFile();
  document.getElementById('btnPublicar').innerHTML = '<svg width="13" height="13" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM6.293 6.707a1 1 0 010-1.414l3-3a1 1 0 011.414 0l3 3a1 1 0 01-1.414 1.414L11 5.414V13a1 1 0 11-2 0V5.414L7.707 6.707a1 1 0 01-1.414 0z" clip-rule="evenodd"/></svg> Publicar';
  document.getElementById('btnPublicar').style.background = '';
  handleHostInput();
}

async function loadInstances() {
  try {
    instances = await apiListInstancias();
  } catch (error) {
    instances = [];
    showToast(error.message, 'error');
  }
  renderInstances();
}

function renderInstances() {
  const empty = document.getElementById('emptyState');
  const table = document.getElementById('instancesTable');
  const tbody = document.getElementById('instancesTableBody');
  const countBadge = document.getElementById('activeCount');

  if (!instances || instances.length === 0) {
    empty.style.display = 'block';
    table.style.display = 'none';
    countBadge.style.display = 'none';
    return;
  }

  empty.style.display = 'none';
  table.style.display = 'table';
  countBadge.style.display = 'inline-flex';
  countBadge.textContent = `${instances.length} Activa${instances.length > 1 ? 's' : ''}`;

  tbody.innerHTML = instances.map((instance) => `
      <tr>
        <td>
          <a href="${instance.domain}" target="_blank" rel="noreferrer" class="domain-link">
            <span class="online-dot"></span>${instance.domain.replace('http://', '')}
          </a>
        </td>
        <td><span class="ip-tag">${instance.ip}</span></td>
        <td><span style="font-size:13px;">${instance.host}</span></td>
        <td><span class="created-date">${formatDateTime(instance.created)}</span></td>
        <td><button class="btn btn-danger" onclick="eliminar('${instance.host}')">Eliminar</button></td>
      </tr>
    `).join('');
}

async function eliminar(host) {
  if (!window.confirm(`¿Eliminar la instancia "${host}"? Esta acción no se puede deshacer.`)) {
    return;
  }

  try {
    await apiEliminar(host);
    showToast(`Instancia "${host}" eliminada.`, 'success');
    await loadInstances();
  } catch (error) {
    showToast(error.message, 'error');
  }
}

document.getElementById('btnAceptar').disabled = true;
handleHostInput();
loadInstances();
