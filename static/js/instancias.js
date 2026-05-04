let instances = [];
let pendingDelete = -1;

function loadInstances() {
  instances = JSON.parse(localStorage.getItem('httpaas_instances') || '[]');
}

function checkQuick() {
  const val = document.getElementById('quickHost').value.trim();
  document.getElementById('btnDeploy').disabled = val.length < 2;
}

function quickDeploy() {
  const host = document.getElementById('quickHost').value.trim();
  if (!host) return;
  const btn = document.getElementById('btnDeploy');
  btn.innerHTML = '<span class="spinner"></span> Desplegando...';
  btn.disabled = true;

  setTimeout(() => {
    const ip = '192.168.10.' + (30 + instances.length) + '/24';
    const now = new Date();
    const dateStr = now.getFullYear() + '-' + pad(now.getMonth()+1) + '-' + pad(now.getDate())
                  + ' ' + pad(now.getHours()) + ':' + pad(now.getMinutes()) + ':' + pad(now.getSeconds());
    instances.push({ host, ip, domain: 'http://' + host + '.cloud.local', created: dateStr });
    localStorage.setItem('httpaas_instances', JSON.stringify(instances));
    render();
    toast('✓ ' + host + '.cloud.local desplegado', 'success');
    btn.innerHTML = '<svg width="13" height="13" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM6.293 6.707a1 1 0 010-1.414l3-3a1 1 0 011.414 0l3 3a1 1 0 01-1.414 1.414L11 5.414V13a1 1 0 11-2 0V5.414L7.707 6.707a1 1 0 01-1.414 0z" clip-rule="evenodd"/></svg> Desplegar Instancia';
    btn.disabled = true;
    document.getElementById('quickHost').value = '';
  }, 1800);
}

function pad(n) { return n < 10 ? '0'+n : n; }

function filterInstances() {
  render();
}

function render() {
  loadInstances();
  const search = document.getElementById('searchInput').value.toLowerCase();
  const filtered = instances.filter(i =>
    i.host.includes(search) || i.ip.includes(search) || i.domain.includes(search)
  );

  document.getElementById('statActive').textContent = instances.length;
  document.getElementById('statIPs').textContent = instances.length;
  document.getElementById('statDNS').textContent = instances.length;

  const badge = document.getElementById('activeCountBadge');
  if (instances.length > 0) {
    badge.style.display = 'inline-flex';
    badge.textContent = instances.length + ' Activa' + (instances.length !== 1 ? 's' : '');
  } else {
    badge.style.display = 'none';
  }

  const wrapper = document.getElementById('tableWrapper');

  if (filtered.length === 0) {
    wrapper.innerHTML = `
      <div class="empty-state">
        <div class="empty-icon">🖥️</div>
        <h3>${instances.length === 0 ? 'No hay instancias activas' : 'Sin resultados'}</h3>
        <p>${instances.length === 0
          ? 'Define un hostname y despliega tu primera instancia usando el formulario de arriba o el flujo completo de despliegues.'
          : 'Ninguna instancia coincide con tu búsqueda.'}</p>
        ${instances.length === 0 ? '<a href="despliegues.html" class="btn btn-primary">Crear primera instancia</a>' : ''}
      </div>`;
    return;
  }

  wrapper.innerHTML = `
    <table>
      <thead>
        <tr>
          <th>Domain</th>
          <th>IP Address</th>
          <th>Host</th>
          <th>Created</th>
          <th style="text-align:right">Action</th>
        </tr>
      </thead>
      <tbody>
        ${filtered.map((inst, i) => `
          <tr>
            <td>
              <div class="domain-cell">
                <span class="online-dot"></span>
                <a href="${inst.domain}" target="_blank" class="domain-link">${inst.domain}</a>
              </div>
            </td>
            <td><span class="ip-mono">${inst.ip}</span></td>
            <td><span class="host-chip">${inst.host}</span></td>
            <td><span class="date-text">${inst.created}</span></td>
            <td style="text-align:right">
              <button class="btn btn-danger" onclick="openModal(${instances.indexOf(inst)})">Eliminar</button>
            </td>
          </tr>
        `).join('')}
      </tbody>
    </table>`;
}

function refreshAll() {
  render();
  toast('Lista actualizada', 'success');
}

// ── Modal ──────────────────────────────────────────────────────────────
function openModal(i) {
  pendingDelete = i;
  document.getElementById('modalMsg').textContent =
    '¿Estás seguro de que deseas eliminar la instancia "' + instances[i].host + '"? ' +
    'Se destruirá la máquina virtual y el registro DNS "' + instances[i].host + '.cloud.local" será removido del servidor Bind9.';
  document.getElementById('modalOverlay').classList.add('open');
}

function closeModal() {
  pendingDelete = -1;
  document.getElementById('modalOverlay').classList.remove('open');
}

function confirmDelete() {
  if (pendingDelete < 0) return;
  const btn = document.getElementById('modalConfirmBtn');
  btn.innerHTML = '<span class="spinner"></span> Eliminando...';
  btn.disabled = true;

  setTimeout(() => {
    const host = instances[pendingDelete].host;
    instances.splice(pendingDelete, 1);
    localStorage.setItem('httpaas_instances', JSON.stringify(instances));
    closeModal();
    render();
    toast('Instancia "' + host + '" eliminada correctamente.', 'error');
    btn.textContent = 'Eliminar definitivamente';
    btn.disabled = false;
  }, 1200);
}

// Close modal on overlay click
document.getElementById('modalOverlay').addEventListener('click', function(e) {
  if (e.target === this) closeModal();
});

// ── Toast ──────────────────────────────────────────────────────────────
function toast(msg, type='success') {
  const container = document.getElementById('toasts');
  const el = document.createElement('div');
  el.className = 'toast ' + type;
  el.textContent = msg;
  container.appendChild(el);
  setTimeout(() => el.remove(), 4000);
}

// Init
render();
