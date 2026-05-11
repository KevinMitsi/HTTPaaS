const HTTPAAS_DOMAIN = 'cloud.local';

async function httpaasRequest(path, options = {}) {
  const response = await fetch(path, options);
  const contentType = response.headers.get('content-type') || '';
  let payload = null;

  if (contentType.includes('application/json')) {
    payload = await response.json();
  } else {
    const text = await response.text();
    payload = text ? { raw: text } : null;
  }

  if (!response.ok) {
    const message = payload && payload.error ? payload.error : `Error HTTP ${response.status}`;
    throw new Error(message);
  }

  return payload;
}

async function apiListInstancias() {
  return httpaasRequest('/api/instancias');
}

async function apiProvisionar(host, file) {
  const formData = new FormData();
  formData.append('host', host);
  formData.append('archivo', file);
  return httpaasRequest('/api/provisionar', {
    method: 'POST',
    body: formData,
  });
}

async function apiEliminar(host) {
  return httpaasRequest(`/api/eliminar/${encodeURIComponent(host)}`, {
    method: 'DELETE',
  });
}

function buildDomain(host) {
  return `http://${host}.${HTTPAAS_DOMAIN}`;
}

function formatDateTime(value) {
  if (!value) {
    return '';
  }
  return value;
}

function showToast(message, type = 'success') {
  const container = document.getElementById('toasts');
  if (!container) {
    return;
  }

  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.textContent = message;
  container.appendChild(toast);

  window.setTimeout(() => toast.remove(), 4000);
}
