// ── State ──────────────────────────────────────────────────────────────
let hostAccepted = false;
let fileReady    = false;
let currentHost  = '';

// Cargar instancias desde localStorage (para demo)
let instances = JSON.parse(localStorage.getItem('httpaas_instances') || '[]');

// ── Host ───────────────────────────────────────────────────────────────
function handleHostInput() {
  const val = document.getElementById('hostInput').value.trim();
  document.getElementById('btnAceptar').disabled = val.length < 2;
}

function aceptarHost() {
  const val = document.getElementById('hostInput').value.trim();
  if (!val) return;
  currentHost = val;
  hostAccepted = true;

  // Enable step 2
  document.getElementById('step2Section').classList.remove('disabled');
  document.getElementById('hostInput').readOnly = true;
  document.getElementById('btnAceptar').textContent = '✓ Aceptado';
  document.getElementById('btnAceptar').style.background = '#16a34a';
  document.getElementById('btnAceptar').disabled = true;

  toast('Host "' + val + '" configurado. Ahora suba el contenido web.', 'success');
}

// ── File ───────────────────────────────────────────────────────────────
function handleFile(input) {
  const file = input.files[0];
  if (!file) return;
  showFile(file);
}

function showFile(file) {
  fileReady = true;
  const sizeMB = (file.size / 1048576).toFixed(1);
  document.getElementById('dropArea').style.display = 'none';
  document.getElementById('fileSelected').classList.add('show');
  document.getElementById('fileName').textContent = file.name;
  document.getElementById('fileSize').textContent = sizeMB + ' MB · Preparado para subir';
  document.getElementById('btnPublicar').disabled = false;
}

function removeFile() {
  fileReady = false;
  document.getElementById('fileInput').value = '';
  document.getElementById('dropArea').style.display = 'block';
  document.getElementById('fileSelected').classList.remove('show');
  document.getElementById('btnPublicar').disabled = true;
}

// Drag & drop
const dropArea = document.getElementById('dropArea');
dropArea.addEventListener('dragover', e => { e.preventDefault(); dropArea.style.borderColor='var(--accent)'; dropArea.style.background='var(--accent-lt)'; });
dropArea.addEventListener('dragleave', () => { dropArea.style.borderColor=''; dropArea.style.background=''; });
dropArea.addEventListener('drop', e => {
  e.preventDefault();
  dropArea.style.borderColor=''; dropArea.style.background='';
  const file = e.dataTransfer.files[0];
  if (file && (file.name.endsWith('.zip') || file.name.endsWith('.tar.gz'))) showFile(file);
  else toast('Solo se aceptan archivos .zip o .tar.gz', 'error');
});

// ── Publish ────────────────────────────────────────────────────────────
function publicar() {
  const btn = document.getElementById('btnPublicar');
  btn.innerHTML = '<span class="spinner"></span> Publicando...';
  btn.disabled = true;

  // Simula llamada al backend Go: POST /api/instancias
  setTimeout(() => {
    const ip = '192.168.10.' + (30 + instances.length);
    const now = new Date();
    const dateStr = now.getFullYear() + '-' + pad(now.getMonth()+1) + '-' + pad(now.getDate())
                  + ' ' + pad(now.getHours()) + ':' + pad(now.getMinutes()) + ':' + pad(now.getSeconds());
    const inst = { host: currentHost, ip: ip + '/24', domain: 'http://' + currentHost + '.cloud.local', created: dateStr };
    instances.push(inst);
    localStorage.setItem('httpaas_instances', JSON.stringify(instances));

    renderInstances();
    toast('✓ ' + currentHost + '.cloud.local desplegado correctamente', 'success');

    // Reset form
    btn.innerHTML = '✓ Publicado';
    btn.style.background = '#16a34a';
    setTimeout(() => resetForm(), 2000);
  }, 2200);
}

function pad(n) { return n < 10 ? '0'+n : n; }

function resetForm() {
  hostAccepted = false; fileReady = false; currentHost = '';
  document.getElementById('hostInput').value = '';
  document.getElementById('hostInput').readOnly = false;
  document.getElementById('btnAceptar').textContent = 'Aceptar';
  document.getElementById('btnAceptar').style.background = '';
  document.getElementById('btnAceptar').disabled = false;
  document.getElementById('step2Section').classList.add('disabled');
  removeFile();
  document.getElementById('btnPublicar').innerHTML = '<svg width="13" height="13" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM6.293 6.707a1 1 0 010-1.414l3-3a1 1 0 011.414 0l3 3a1 1 0 01-1.414 1.414L11 5.414V13a1 1 0 11-2 0V5.414L7.707 6.707a1 1 0 01-1.414 0z" clip-rule="evenodd"/></svg> Publicar';
  document.getElementById('btnPublicar').style.background = '';
}

// ── Instances ──────────────────────────────────────────────────────────
function loadInstances() { renderInstances(); }

function renderInstances() {
  instances = JSON.parse(localStorage.getItem('httpaas_instances') || '[]');
  const empty = document.getElementById('emptyState');
  const table = document.getElementById('instancesTable');
  const tbody = document.getElementById('instancesTableBody');
  const countBadge = document.getElementById('activeCount');

  if (instances.length === 0) {
    empty.style.display = 'block';
    table.style.display = 'none';
    countBadge.style.display = 'none';
    return;
  }

  empty.style.display = 'none';
  table.style.display = 'table';
  countBadge.style.display = 'inline-flex';
  countBadge.textContent = instances.length + ' Activa' + (instances.length > 1 ? 's' : '');

  tbody.innerHTML = instances.map((inst, i) => `
      <tr>
        <td>
          <a href="${inst.domain}" target="_blank" class="domain-link">
            <span class="online-dot"></span>${inst.domain.replace('http://','')}
          </a>
        </td>
        <td><span class="ip-tag">${inst.ip}</span></td>
        <td><span style="font-size:13px;">${inst.host}</span></td>
        <td><span class="created-date">${inst.created}</span></td>
        <td><button class="btn btn-danger" onclick="eliminar(${i})">Eliminar</button></td>
      </tr>
    `).join('');
}

function eliminar(i) {
  if (!confirm('¿Eliminar la instancia "' + instances[i].host + '"? Esta acción no se puede deshacer.')) return;
  const host = instances[i].host;
  instances.splice(i, 1);
  localStorage.setItem('httpaas_instances', JSON.stringify(instances));
  renderInstances();
  toast('Instancia "' + host + '" eliminada.', 'error');
}

// ── Toast ──────────────────────────────────────────────────────────────
function toast(msg, type='success') {
  const container = document.getElementById('toasts');
  const el = document.createElement('div');
  el.className = 'toast ' + type;
  el.innerHTML = (type === 'success' ? '✓' : '✕') + ' ' + msg;
  container.appendChild(el);
  setTimeout(() => el.remove(), 4000);
}

// ── Init ───────────────────────────────────────────────────────────────
document.getElementById('btnAceptar').disabled = true;
renderInstances();
