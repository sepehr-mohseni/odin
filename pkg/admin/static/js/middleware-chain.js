// Middleware Chain Management
let middlewareChain = [];
let middlewareRoutes = [];
let editingMiddlewareRoutes = [];
let availablePlugins = [];

// Load middleware chain on page load
document.addEventListener('DOMContentLoaded', () => {
  loadMiddlewareChain();
  loadAvailablePlugins();
  initializeEventListeners();
});

// Initialize event listeners
function initializeEventListeners() {
  // Stats button
  document.getElementById('statsBtn').addEventListener('click', toggleStats);
  
  // Reload all button
  document.getElementById('reloadAllBtn').addEventListener('click', reloadAllMiddleware);
  
  // Add middleware buttons
  document.getElementById('addMiddlewareBtn').addEventListener('click', showRegisterModal);
  document.getElementById('addFirstMiddleware')?.addEventListener('click', showRegisterModal);
  
  // Register middleware
  document.getElementById('registerBtn').addEventListener('click', registerMiddleware);
  
  // Priority slider sync
  const prioritySlider = document.getElementById('middlewarePrioritySlider');
  const priorityInput = document.getElementById('middlewarePriority');
  prioritySlider.addEventListener('input', (e) => {
    priorityInput.value = e.target.value;
  });
  priorityInput.addEventListener('input', (e) => {
    prioritySlider.value = e.target.value;
  });
  
  // Edit priority slider sync
  const editPrioritySlider = document.getElementById('editMiddlewarePrioritySlider');
  const editPriorityInput = document.getElementById('editMiddlewarePriority');
  editPrioritySlider.addEventListener('input', (e) => {
    editPriorityInput.value = e.target.value;
  });
  editPriorityInput.addEventListener('input', (e) => {
    editPrioritySlider.value = e.target.value;
  });
  
  // Route management
  document.getElementById('addRouteBtn').addEventListener('click', addRoute);
  document.getElementById('routeInput').addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addRoute();
    }
  });
  
  document.getElementById('editAddRouteBtn').addEventListener('click', addEditRoute);
  document.getElementById('editRouteInput').addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addEditRoute();
    }
  });
  
  // Save middleware changes
  document.getElementById('saveMiddlewareBtn').addEventListener('click', saveMiddlewareChanges);
}

// Load middleware chain from API
async function loadMiddlewareChain() {
  try {
    const response = await OdinAPI.get('/admin/api/middleware/chain');
    middlewareChain = response.chain || [];
    renderMiddlewareChain();
  } catch (error) {
    showToast('Failed to load middleware chain: ' + error.message, 'danger');
  }
}

// Load available plugins for registration
async function loadAvailablePlugins() {
  try {
    const response = await OdinAPI.get('/admin/api/plugins');
    availablePlugins = response.filter(p => p.pluginType === 'middleware' && p.loaded);
    populatePluginSelect();
  } catch (error) {
    console.error('Failed to load plugins:', error);
  }
}

// Populate plugin select dropdown
function populatePluginSelect() {
  const select = document.getElementById('middlewareName');
  select.innerHTML = '<option value="">Select a loaded plugin...</option>';
  
  availablePlugins.forEach(plugin => {
    const option = document.createElement('option');
    option.value = plugin.name;
    option.textContent = `${plugin.name} (v${plugin.version})`;
    select.appendChild(option);
  });
}

// Render middleware chain
function renderMiddlewareChain() {
  const container = document.getElementById('middlewareChain');
  const emptyState = document.getElementById('emptyState');
  
  if (middlewareChain.length === 0) {
    container.style.display = 'none';
    emptyState.style.display = 'block';
    return;
  }
  
  container.style.display = 'block';
  emptyState.style.display = 'none';
  container.innerHTML = '';
  
  middlewareChain.forEach((middleware, index) => {
    const item = createMiddlewareItem(middleware, index);
    container.appendChild(item);
  });
  
  // Render phase-specific chains
  renderPhaseChain('pre-auth', 'preAuthChain');
  renderPhaseChain('post-auth', 'postAuthChain');
  renderPhaseChain('pre-route', 'preRouteChain');
  renderPhaseChain('post-route', 'postRouteChain');
}

// Render phase-specific chain
function renderPhaseChain(phase, containerId) {
  const container = document.getElementById(containerId);
  const filtered = middlewareChain.filter(m => m.phase === phase);
  
  if (filtered.length === 0) {
    container.innerHTML = '<p class="text-muted">No middleware in this phase</p>';
    return;
  }
  
  container.innerHTML = '';
  filtered.forEach((middleware, index) => {
    const item = createMiddlewareItem(middleware, index);
    container.appendChild(item);
  });
}

// Create middleware item element
function createMiddlewareItem(middleware, index) {
  const div = document.createElement('div');
  div.className = 'middleware-item';
  div.draggable = true;
  div.dataset.index = index;
  div.dataset.name = middleware.name;
  
  // Drag handle
  const dragHandle = `<i class="bi bi-grip-vertical middleware-drag-handle"></i>`;
  
  // Priority badge
  const priorityBadge = `<span class="middleware-priority">Priority: ${middleware.priority}</span>`;
  
  // Phase badge
  let phaseBadge = '';
  if (middleware.phase) {
    phaseBadge = `<span class="middleware-phase phase-${middleware.phase}">${formatPhase(middleware.phase)}</span>`;
  }
  
  // Routes badges
  let routesBadges = '';
  if (middleware.routes && middleware.routes.length > 0) {
    routesBadges = '<div class="middleware-routes">';
    middleware.routes.forEach(route => {
      const badgeClass = route === '*' ? 'route-badge global' : 'route-badge';
      routesBadges += `<span class="${badgeClass}">${route === '*' ? 'Global (*)' : route}</span>`;
    });
    routesBadges += '</div>';
  } else {
    routesBadges = '<div class="middleware-routes"><span class="route-badge global">Global (*)</span></div>';
  }
  
  // Status badges
  const enabledBadge = middleware.enabled 
    ? '<span class="badge bg-success">Enabled</span>' 
    : '<span class="badge bg-secondary">Disabled</span>';
  
  const loadedBadge = middleware.loaded 
    ? '<span class="badge bg-primary">Loaded</span>' 
    : '<span class="badge bg-warning">Not Loaded</span>';
  
  div.innerHTML = `
    ${priorityBadge}
    <div class="middleware-header">
      ${dragHandle}
      <div class="flex-grow-1">
        <h5 class="middleware-name">${middleware.name}</h5>
        ${middleware.version ? `<span class="middleware-version">v${middleware.version}</span>` : ''}
      </div>
    </div>
    ${middleware.description ? `<p class="middleware-description">${middleware.description}</p>` : ''}
    <div class="mb-2">
      ${phaseBadge}
      ${enabledBadge}
      ${loadedBadge}
    </div>
    ${routesBadges}
    <div class="middleware-actions">
      <button class="btn btn-sm btn-outline-primary" onclick="editMiddleware('${middleware.name}')">
        <i class="bi bi-pencil"></i> Edit
      </button>
      <button class="btn btn-sm btn-outline-info" onclick="testMiddleware('${middleware.name}')">
        <i class="bi bi-play-circle"></i> Test
      </button>
      <button class="btn btn-sm btn-outline-success" onclick="getMiddlewareHealth('${middleware.name}')">
        <i class="bi bi-heart-pulse"></i> Health
      </button>
      <button class="btn btn-sm btn-outline-danger" onclick="unregisterMiddleware('${middleware.name}')">
        <i class="bi bi-x-circle"></i> Unregister
      </button>
    </div>
  `;
  
  // Drag and drop event listeners
  div.addEventListener('dragstart', handleDragStart);
  div.addEventListener('dragover', handleDragOver);
  div.addEventListener('drop', handleDrop);
  div.addEventListener('dragend', handleDragEnd);
  
  return div;
}

// Drag and drop handlers
let draggedElement = null;

function handleDragStart(e) {
  draggedElement = this;
  this.classList.add('dragging');
  e.dataTransfer.effectAllowed = 'move';
  e.dataTransfer.setData('text/html', this.innerHTML);
}

function handleDragOver(e) {
  if (e.preventDefault) {
    e.preventDefault();
  }
  e.dataTransfer.dropEffect = 'move';
  
  if (this !== draggedElement) {
    this.classList.add('drag-over');
  }
  return false;
}

function handleDrop(e) {
  if (e.stopPropagation) {
    e.stopPropagation();
  }
  
  if (draggedElement !== this) {
    const draggedIndex = parseInt(draggedElement.dataset.index);
    const targetIndex = parseInt(this.dataset.index);
    
    // Reorder the chain
    const draggedItem = middlewareChain[draggedIndex];
    middlewareChain.splice(draggedIndex, 1);
    middlewareChain.splice(targetIndex, 0, draggedItem);
    
    // Update priorities based on new order
    updatePrioritiesAfterReorder();
  }
  
  this.classList.remove('drag-over');
  return false;
}

function handleDragEnd(e) {
  this.classList.remove('dragging');
  document.querySelectorAll('.middleware-item').forEach(item => {
    item.classList.remove('drag-over');
  });
}

// Update priorities after reorder
async function updatePrioritiesAfterReorder() {
  const order = middlewareChain.map((middleware, index) => ({
    name: middleware.name,
    priority: index * 10 // Space them out by 10
  }));
  
  try {
    await OdinAPI.post('/admin/api/middleware/chain/reorder', { order });
    showToast('Middleware chain reordered successfully', 'success');
    loadMiddlewareChain(); // Reload to get updated data
  } catch (error) {
    showToast('Failed to reorder middleware chain: ' + error.message, 'danger');
    loadMiddlewareChain(); // Reload to restore previous order
  }
}

// Toggle statistics display
async function toggleStats() {
  const statsCards = document.getElementById('statsCards');
  const isVisible = statsCards.style.display !== 'none';
  
  if (isVisible) {
    statsCards.style.display = 'none';
  } else {
    await loadStats();
    statsCards.style.display = 'flex';
  }
}

// Load statistics
async function loadStats() {
  try {
    const stats = await OdinAPI.get('/admin/api/middleware/chain/stats');
    document.getElementById('statTotal').textContent = stats.totalMiddlewares;
    document.getElementById('statActive').textContent = stats.activeInChain;
    document.getElementById('statGlobal').textContent = stats.globalMiddlewares;
    document.getElementById('statRouteSpecific').textContent = stats.routeSpecific;
  } catch (error) {
    showToast('Failed to load statistics: ' + error.message, 'danger');
  }
}

// Show register middleware modal
function showRegisterModal() {
  middlewareRoutes = [];
  renderRoutesList();
  document.getElementById('registerMiddlewareForm').reset();
  document.getElementById('middlewarePriority').value = 500;
  document.getElementById('middlewarePrioritySlider').value = 500;
  new bootstrap.Modal(document.getElementById('registerMiddlewareModal')).show();
}

// Add route to list
function addRoute() {
  const input = document.getElementById('routeInput');
  const route = input.value.trim();
  
  if (route && !middlewareRoutes.includes(route)) {
    middlewareRoutes.push(route);
    renderRoutesList();
    input.value = '';
  }
}

// Render routes list
function renderRoutesList() {
  const container = document.getElementById('routesList');
  container.innerHTML = '';
  
  middlewareRoutes.forEach((route, index) => {
    const tag = document.createElement('span');
    tag.className = 'route-tag';
    tag.innerHTML = `
      ${route}
      <span class="remove-route" onclick="removeRoute(${index})">×</span>
    `;
    container.appendChild(tag);
  });
}

// Remove route from list
function removeRoute(index) {
  middlewareRoutes.splice(index, 1);
  renderRoutesList();
}

// Add route to edit list
function addEditRoute() {
  const input = document.getElementById('editRouteInput');
  const route = input.value.trim();
  
  if (route && !editingMiddlewareRoutes.includes(route)) {
    editingMiddlewareRoutes.push(route);
    renderEditRoutesList();
    input.value = '';
  }
}

// Render edit routes list
function renderEditRoutesList() {
  const container = document.getElementById('editRoutesList');
  container.innerHTML = '';
  
  editingMiddlewareRoutes.forEach((route, index) => {
    const tag = document.createElement('span');
    tag.className = 'route-tag';
    tag.innerHTML = `
      ${route}
      <span class="remove-route" onclick="removeEditRoute(${index})">×</span>
    `;
    container.appendChild(tag);
  });
}

// Remove route from edit list
function removeEditRoute(index) {
  editingMiddlewareRoutes.splice(index, 1);
  renderEditRoutesList();
}

// Register middleware
async function registerMiddleware() {
  const name = document.getElementById('middlewareName').value;
  const priority = parseInt(document.getElementById('middlewarePriority').value);
  const phase = document.getElementById('middlewarePhase').value;
  
  if (!name) {
    showToast('Please select a middleware plugin', 'warning');
    return;
  }
  
  const data = {
    priority,
    routes: middlewareRoutes.length > 0 ? middlewareRoutes : ['*'],
    phase
  };
  
  try {
    await OdinAPI.post(`/admin/api/middleware/${name}/register`, data);
    showToast(`Middleware ${name} registered successfully`, 'success');
    bootstrap.Modal.getInstance(document.getElementById('registerMiddlewareModal')).hide();
    loadMiddlewareChain();
  } catch (error) {
    showToast('Failed to register middleware: ' + error.message, 'danger');
  }
}

// Edit middleware
function editMiddleware(name) {
  const middleware = middlewareChain.find(m => m.name === name);
  if (!middleware) return;
  
  document.getElementById('editMiddlewareName').value = name;
  document.getElementById('editMiddlewareNameDisplay').value = name;
  document.getElementById('editMiddlewarePriority').value = middleware.priority;
  document.getElementById('editMiddlewarePrioritySlider').value = middleware.priority;
  document.getElementById('editMiddlewarePhase').value = middleware.phase || '';
  
  editingMiddlewareRoutes = middleware.routes ? [...middleware.routes] : [];
  renderEditRoutesList();
  
  new bootstrap.Modal(document.getElementById('editMiddlewareModal')).show();
}

// Save middleware changes
async function saveMiddlewareChanges() {
  const name = document.getElementById('editMiddlewareName').value;
  const priority = parseInt(document.getElementById('editMiddlewarePriority').value);
  const phase = document.getElementById('editMiddlewarePhase').value;
  
  try {
    // Update priority
    await OdinAPI.put(`/admin/api/middleware/${name}/priority`, { priority });
    
    // Update routes
    await OdinAPI.put(`/admin/api/middleware/${name}/routes`, { 
      routes: editingMiddlewareRoutes.length > 0 ? editingMiddlewareRoutes : ['*'] 
    });
    
    // Update phase
    if (phase) {
      await OdinAPI.put(`/admin/api/middleware/${name}/phase`, { phase });
    }
    
    showToast(`Middleware ${name} updated successfully`, 'success');
    bootstrap.Modal.getInstance(document.getElementById('editMiddlewareModal')).hide();
    loadMiddlewareChain();
  } catch (error) {
    showToast('Failed to update middleware: ' + error.message, 'danger');
  }
}

// Test middleware
async function testMiddleware(name) {
  try {
    const result = await OdinAPI.post(`/admin/api/middleware/${name}/test`, {
      testPath: '/api/test',
      testMethod: 'GET',
      testHeaders: {}
    });
    showToast(`Middleware ${name} test: ${result.testPassed ? 'PASSED' : 'FAILED'}`, 
              result.testPassed ? 'success' : 'danger');
  } catch (error) {
    showToast('Failed to test middleware: ' + error.message, 'danger');
  }
}

// Get middleware health
async function getMiddlewareHealth(name) {
  try {
    const health = await OdinAPI.get(`/admin/api/middleware/${name}/health`);
    const status = health.healthy ? 'Healthy' : 'Unhealthy';
    const color = health.healthy ? 'success' : 'danger';
    showToast(`${name}: ${status} (v${health.version})`, color);
  } catch (error) {
    showToast('Failed to get middleware health: ' + error.message, 'danger');
  }
}

// Unregister middleware
async function unregisterMiddleware(name) {
  if (!await confirm(`Unregister middleware "${name}" from the chain?`, 'Unregister Middleware')) {
    return;
  }
  
  try {
    await OdinAPI.delete(`/admin/api/middleware/${name}/unregister`);
    showToast(`Middleware ${name} unregistered successfully`, 'success');
    loadMiddlewareChain();
  } catch (error) {
    showToast('Failed to unregister middleware: ' + error.message, 'danger');
  }
}

// Reload all middleware
async function reloadAllMiddleware() {
  if (!await confirm('Reload all middleware from database? This may briefly interrupt service.', 'Reload All Middleware')) {
    return;
  }
  
  try {
    const result = await OdinAPI.post('/admin/api/middleware/reload-all', {});
    showToast(`Reloaded ${result.reloaded} middleware (${result.errors} errors)`, 
              result.errors > 0 ? 'warning' : 'success');
    loadMiddlewareChain();
  } catch (error) {
    showToast('Failed to reload middleware: ' + error.message, 'danger');
  }
}

// Helper: Format phase name
function formatPhase(phase) {
  const names = {
    'pre-auth': 'Pre-Auth',
    'post-auth': 'Post-Auth',
    'pre-route': 'Pre-Route',
    'post-route': 'Post-Route'
  };
  return names[phase] || phase;
}
