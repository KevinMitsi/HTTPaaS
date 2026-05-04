let hostAccepted = false;
let currentHost  = '';
let fileObj      = null;
let instances    = JSON.parse(localStorage.getItem('httpaas_instances') || '[]');

function checkHost() {
  const val = document.getElementById('hostInput').value.trim();
  document.getElementById('domainPreview').textContent = val ? val + '.cloud.local' : '‹hostname›.cloud.local';
  document.getElementById('btnAceptar').disabled = val.length < 2;
}

function aceptarHost() {
  const val = document.getElementById('hostInput').value.trim();
  if (!val) return;
  currentHost = val;
  hostAccepted = true;

  // Mark step 1 done
  document.getElementById('card1').classList.remove('active');
  document.getElementById('card1').classList.add('done');
  document.getElementById('badge1').classList.replace('active','done');
  document.getElementById('badge1').textContent = '✓';
  document.getElementById('status1').style.display = 'block';
  document.getElementById('body1').classList.add('collapsed');
  document.getElementById('hostInput').readOnly = true;

  // Enable step 2
  const card2 = document.getElementById('card2');
  card2.style.opacity = '1';
  card2.style.pointerEvents = 'auto';
  card2.classList.add('active');
  document.getElementById('badge2').classList.replace('pending','active');
  document.getElementById('body2').classList.remove('collapsed');

  toast('Host "' + val + '.cloud.local" configurado', 'success');
  renderRightPanel();
}

function resetStep1() {
  currentHost = ''; hostAccepted = false;
  document.getElementById('card1').classList.add('active');
  document.getElementById('card1').classList.remove('done');
  document.getElementById('badge1').classList.replace('done','active');
  document.getElementById('badge1').textContent = '1';
  document.getElementById('status1').style.display = 'none';
  document.getElementById('body1').classList.remove('collapsed');
  document.getElementById('hostInput').readOnly = false;
  document.getElementById('hostInput').value = '';

  document.getElementById('card2').style.opacity = '.5';
  document.getElementById('card2').style.pointerEvents = 'none';
  document.getElementById('card2').classList.remove('active');
  document.getElementById('badge2').classList.replace('active','pending');
  document.getElementById('body2').classList.add('collapsed');
  removeFile();
}

function handleFile(input) {
  const file = input.files[0];
  if (!file) return;
  fileObj = file;
  const sizeMB = (file.size / 1048576).toFixed(1);
  document.getElementById('dropArea').style.display = 'none';
  document.getElementById('fileSelected').classList.add('show');
  document.getElementById('fscName').textContent = file.name;
  document.getElementById('fscSize').textContent = sizeMB + ' MB · Preparado para subir';
  document.getElementById('btnPublicar').disabled = false;
}

function removeFile() {
  fileObj = null;
  document.getElementById('fileInput').value = '';
  document.getElementById('dropArea').style.display = 'block';
  document.getElementById('fileSelected').classList.remove('show');
  document.getElementById('btnPublicar').disabled = true;
}

// Drag & drop
const dropArea = document.getElementById('dropArea');
dropArea.addEventListener('dragover', e => { e.preventDefault(); dropArea.style.borderColor='var(--accent)'; });
dropArea.addEventListener('dragleave', () => { dropArea.style.borderColor=''; });
dropArea.addEventListener('drop', e => {
  e.preventDefault(); dropArea.style.borderColor='';
  const file = e.dataTransfer.files[0];
  if (file) handleFile({ files: [file] });
});

function publicar() {
  const btn = document.getElementById('btnPublicar');
  const log = document.getElementById('progressLog');
  btn.innerHTML = '<span class="spinner"></span> Desplegando...';
  btn.disabled = true;
  document.getElementById('fscRemove') && (document.getElementById('fscRemove').disabled = true);

  // Show progress log
  log.style.display = 'block';
  log.innerHTML = '';

  const steps = [
    '→ Clonando disco multiconexión ApacheServer.vdi...',
    '→ Creando máquina virtual "' + currentHost + '" en VirtualBox...',
    '→ Configurando red interna (192.168.10.x/24)...',
    '→ Iniciando VM y esperando SSH...',
    '→ Registrando host DNS: ' + currentHost + ' → 192.168.10.' + (30 + instances.length),
    '→ Transfiriendo contenido web (' + document.getElementById('fscName').textContent + ')...',
    '→ Extrayendo archivos en /var/www/html/' + currentHost + '/',
    '✓ Despliegue completado. Instancia en línea.',
  ];

  let i = 0;
  const interval = setInterval(() => {
    if (i >= steps.length) {
      clearInterval(interval);
      finalizarDespliegue();
      return;
    }
    const line = document.createElement('div');
    line.className = 'log-line';
    line.textContent = steps[i];
    log.appendChild(line);
    log.scrollTop = log.scrollHeight;
    i++;
  }, 380);
}

function finalizarDespliegue() {
  const ip = '192.168.10.' + (30 + instances.length) + '/24';
  const now = new Date();
  const dateStr = now.getFullYear() + '-' + pad(now.getMonth()+1) + '-' + pad(now.getDate())
                + ' ' + pad(now.getHours()) + ':' + pad(now.getMinutes()) + ':' + pad(now.getSeconds());
  instances.push({ host: currentHost, ip, domain: 'http://' + currentHost + '.cloud.local', created: dateStr });
  localStorage.setItem('httpaas_instances', JSON.stringify(instances));

  toast('✓ ' + currentHost + '.cloud.local está en línea', 'success');
  renderRightPanel();

  // Mark step 2 done
  document.getElementById('badge2').classList.replace('active','done');
  document.getElementById('badge2').textContent = '✓';
  document.getElementById('card2').classList.remove('active');
  document.getElementById('card2').classList.add('done');
  document.getElementById('status2').style.display = 'block';

  setTimeout(() => {
    if (confirm('¿Deseas crear otro despliegue?')) {
      location.reload();
    } else {
      window.location.href = 'instancias.html';
    }
  }, 1500);
}

function pad(n) { return n < 10 ? '0'+n : n; }

function renderRightPanel() {
  instances = JSON.parse(localStorage.getItem('httpaas_instances') || '[]');
  const panel = document.getElementById('rightPanel');
  if (instances.length === 0) {
    panel.innerHTML = `
      <div class="panel-empty">
        <div class="panel-empty-icon">🖥️</div>
        <p>No hay instancias activas.<br>Las instancias aparecerán aquí una vez finalice la publicación.</p>
      </div>`;
    return;
  }
  panel.innerHTML = instances.map(inst => `
    <div class="instance-item">
      <span class="online-dot"></span>
      <a href="${inst.domain}" target="_blank" class="inst-domain"> ${inst.domain}</a>
      <div class="inst-meta">
        <span>${inst.ip}</span>
        <span>·</span>
        <span>${inst.host}</span>
      </div>
    </div>
  `).join('');
}

function toast(msg, type='success') {
  const container = document.getElementById('toasts');
  const el = document.createElement('div');
  el.className = 'toast ' + type;
  el.textContent = msg;
  container.appendChild(el);
  setTimeout(() => el.remove(), 4000);
}

// Init
renderRightPanel();
