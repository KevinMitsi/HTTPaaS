let instances = [];
let searchTerm = '';
let pendingDeleteHost = '';

function checkQuick() {
  const value = document.getElementById('quickHost').value.trim();
  document.getElementById('btnDeploy').disabled = value.length < 2;
}

function quickDeploy() {
  const host = document.getElementById('quickHost').value.trim();
  if (!host) {
    return;
  }
  window.location.href = `despliegues.html?host=${encodeURIComponent(host)}`;
}

async function loadInstances() {
  try {
    instances = await apiListInstancias();
    render();
  } catch (error) {
    instances = [];
    renderEmptyState(error.message);
    showToast(error.message, 'error');
  }
}

function filterInstances() {
  searchTerm = document.getElementById('searchInput').value.trim().toLowerCase();
  render();
}

function render() {
  const search = searchTerm;
  const filtered = instances.filter((instance) =>
    instance.host.toLowerCase().includes(search) ||
    instance.ip.toLowerCase().includes(search) ||
    instance.domain.toLowerCase().includes(search)
  );

  document.getElementById('statActive').textContent = instances.length;
  document.getElementById('statIPs').textContent = instances.length;
  document.getElementById('statDNS').textContent = instances.length;

  const badge = document.getElementById('activeCountBadge');
  if (instances.length > 0) {
    badge.style.display = 'inline-flex';
    badge.textContent = `${instances.length} Activa${instances.length !== 1 ? 's' : ''}`;
  } else {
    badge.style.display = 'none';
  }

  const wrapper = document.getElementById('tableWrapper');
  if (filtered.length === 0) {
    renderEmptyState(instances.length === 0 ? 'No hay instancias activas' : 'Sin resultados');
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
        ${filtered.map((instance) => `
          <tr>
            <td>
              <div class="domain-cell">
                <span class="online-dot"></span>
                <a href="${instance.domain}" target="_blank" rel="noreferrer" class="domain-link">${instance.domain}</a>
              </div>
            </td>
            <td><span class="ip-mono">${instance.ip}</span></td>
            <td><span class="host-chip">${instance.host}</span></td>
            <td><span class="date-text">${formatDateTime(instance.created)}</span></td>
            <td style="text-align:right">
              <button class="btn btn-danger" onclick="openModal('${instance.host}')">Eliminar</button>
            </td>
          </tr>
        `).join('')}
      </tbody>
    </table>`;
}

function renderEmptyState(message) {
  const wrapper = document.getElementById('tableWrapper');
  wrapper.innerHTML = `
    <div class="empty-state">
      <div class="empty-icon">🖥️</div>
      <h3>${message === 'Sin resultados' ? 'Sin resultados' : 'No hay instancias activas'}</h3>
      <p>${message === 'Sin resultados'
        ? 'Ninguna instancia coincide con tu búsqueda.'
        : 'Define un hostname y despliega tu primera instancia usando el flujo completo de despliegues.'}</p>
      ${message === 'No hay instancias activas' ? '<a href="despliegues.html" class="btn btn-primary">Crear primera instancia</a>' : ''}
    </div>`;
}

async function refreshAll() {
  await loadInstances();
  showToast('Lista actualizada', 'success');
}

function openModal(host) {
  pendingDeleteHost = host;
  const instance = instances.find((item) => item.host === host);
  document.getElementById('modalMsg').textContent =
    `¿Estás seguro de que deseas eliminar la instancia "${host}"? Se destruirá la máquina virtual y el registro DNS "${host}.cloud.local" será removido del servidor Bind9.`;
  if (!instance) {
    showToast('No se encontró la instancia seleccionada', 'error');
    return;
  }
  document.getElementById('modalOverlay').classList.add('open');
}

function closeModal() {
  pendingDeleteHost = '';
  document.getElementById('modalOverlay').classList.remove('open');
  const btn = document.getElementById('modalConfirmBtn');
  btn.textContent = 'Eliminar definitivamente';
  btn.disabled = false;
}

async function confirmDelete() {
  if (!pendingDeleteHost) {
    return;
  }

  const btn = document.getElementById('modalConfirmBtn');
  btn.innerHTML = '<span class="spinner"></span> Eliminando...';
  btn.disabled = true;

  try {
    await apiEliminar(pendingDeleteHost);
    showToast(`Instancia "${pendingDeleteHost}" eliminada correctamente.`, 'success');
    closeModal();
    await loadInstances();
  } catch (error) {
    showToast(error.message, 'error');
    btn.textContent = 'Eliminar definitivamente';
    btn.disabled = false;
  }
}

document.getElementById('modalOverlay').addEventListener('click', function (event) {
  if (event.target === this) {
    closeModal();
  }
});

loadInstances();
