/**
 * Odin API Gateway - Plugins Module
 * Plugin-specific JavaScript functionality
 */

const OdinPlugins = (() => {
  'use strict';

  // Template descriptions
  const TEMPLATES = {
    auth: 'Authentication middleware with token validation and permission checking',
    ratelimit: 'Rate limiting middleware with per-IP/user request limits',
    logging: 'Request/response logging middleware with duration tracking',
    transform: 'Request/response transformation middleware for modifying data',
    cache: 'Response caching middleware with TTL and SHA256-based keys',
    custom: 'Blank custom plugin template for building your own functionality',
  };

  /**
   * Initialize plugin form handlers
   */
  function initPluginForm() {
    const form = document.getElementById('plugin-form');
    if (!form) return;

    // Source type selection
    const sourceRadios = document.querySelectorAll('input[name="sourceType"]');
    sourceRadios.forEach((radio) => {
      radio.addEventListener('change', handleSourceTypeChange);
    });

    // Template selection
    const templateSelect = document.getElementById('templateType');
    if (templateSelect) {
      templateSelect.addEventListener('change', handleTemplateChange);
    }

    // Plugin type selection
    const typeRadios = document.querySelectorAll('input[name="pluginType"]');
    typeRadios.forEach((radio) => {
      radio.addEventListener('change', handlePluginTypeChange);
    });

    // Form submission
    form.addEventListener('submit', handlePluginSubmit);
  }

  /**
   * Handle source type change (file, path, template)
   */
  function handleSourceTypeChange(event) {
    const sourceType = event.target.value;
    const sections = document.querySelectorAll('.source-section');

    // Hide all sections
    sections.forEach((section) => {
      section.style.display = 'none';
    });

    // Show selected section
    if (sourceType === 'file') {
      document.getElementById('fileUploadSection').style.display = 'block';
    } else if (sourceType === 'path') {
      document.getElementById('pathSection').style.display = 'block';
    } else if (sourceType === 'template') {
      document.getElementById('templateSection').style.display = 'block';
    }
  }

  /**
   * Handle template selection change
   */
  function handleTemplateChange(event) {
    const template = event.target.value;
    const templateInfo = document.getElementById('templateInfo');
    const templateDescription = document.getElementById('templateDescription');

    if (template && TEMPLATES[template]) {
      templateDescription.textContent = TEMPLATES[template];
      templateInfo.style.display = 'block';
    } else {
      templateInfo.style.display = 'none';
    }
  }

  /**
   * Handle plugin type change (hooks vs middleware)
   */
  function handlePluginTypeChange(event) {
    const pluginType = event.target.value;
    const hooksSection = document.getElementById('hooksSection');
    const middlewareInfo = document.getElementById('middlewareInfo');

    if (pluginType === 'hooks') {
      if (hooksSection) hooksSection.style.display = 'block';
      if (middlewareInfo) middlewareInfo.style.display = 'none';
    } else if (pluginType === 'middleware') {
      if (hooksSection) hooksSection.style.display = 'none';
      if (middlewareInfo) middlewareInfo.style.display = 'block';
    }
  }

  /**
   * Handle plugin form submission
   */
  async function handlePluginSubmit(event) {
    event.preventDefault();

    const form = event.target;
    const formData = new FormData(form);
    const sourceType = formData.get('sourceType');

    try {
      OdinUI.showLoading('#plugin-form', 'Creating plugin...');

      if (sourceType === 'file') {
        await uploadPluginFile(formData);
      } else if (sourceType === 'path') {
        await createPluginFromPath(formData);
      } else if (sourceType === 'template') {
        await buildPluginFromTemplate(formData);
      }

      OdinUI.showToast('Plugin created successfully!', 'success');
      setTimeout(() => {
        window.location.href = '/admin/plugins';
      }, 1000);
    } catch (error) {
      OdinUI.showError('#plugin-form', error.message);
      console.error('Plugin creation failed:', error);
    }
  }

  /**
   * Upload plugin .so file
   */
  async function uploadPluginFile(formData) {
    const file = formData.get('pluginFile');
    if (!file || file.size === 0) {
      throw new Error('Please select a .so file to upload');
    }

    const metadata = {
      name: formData.get('name'),
      version: formData.get('version'),
      description: formData.get('description') || '',
      pluginType: formData.get('pluginType'),
      hooks: formData.getAll('hooks'),
      enabled: formData.get('enabled') === 'on',
      config: parseConfig(formData.get('config')),
    };

    const uploadData = new FormData();
    uploadData.append('file', file);
    uploadData.append('metadata', JSON.stringify(metadata));

    return OdinAPI.plugins.upload(uploadData);
  }

  /**
   * Create plugin from path
   */
  async function createPluginFromPath(formData) {
    const data = {
      name: formData.get('name'),
      version: formData.get('version'),
      description: formData.get('description') || '',
      binary_path: formData.get('binaryPath'),
      plugin_type: formData.get('pluginType'),
      hooks: formData.getAll('hooks'),
      enabled: formData.get('enabled') === 'on',
      config: parseConfig(formData.get('config')),
    };

    return OdinAPI.plugins.create(data);
  }

  /**
   * Build plugin from template
   */
  async function buildPluginFromTemplate(formData) {
    const data = {
      name: formData.get('name'),
      version: formData.get('version'),
      description: formData.get('description') || '',
      template: formData.get('templateType'),
      pluginType: formData.get('pluginType'),
      hooks: formData.getAll('hooks'),
      enabled: formData.get('enabled') === 'on',
      config: parseConfig(formData.get('config')),
    };

    return OdinAPI.plugins.build(data);
  }

  /**
   * Parse configuration string (JSON)
   */
  function parseConfig(configStr) {
    if (!configStr || configStr.trim() === '') {
      return {};
    }

    try {
      return JSON.parse(configStr);
    } catch (error) {
      throw new Error(`Invalid JSON configuration: ${error.message}`);
    }
  }

  /**
   * Toggle plugin enabled/disabled
   */
  async function togglePlugin(name, currentlyEnabled) {
    try {
      if (currentlyEnabled) {
        await OdinAPI.plugins.disable(name);
        OdinUI.showToast(`Plugin "${name}" disabled`, 'success');
      } else {
        await OdinAPI.plugins.enable(name);
        OdinUI.showToast(`Plugin "${name}" enabled`, 'success');
      }
      // Reload page to update UI
      setTimeout(() => location.reload(), 500);
    } catch (error) {
      OdinUI.showToast(`Failed to toggle plugin: ${error.message}`, 'danger');
    }
  }

  /**
   * Delete plugin with confirmation
   */
  async function deletePlugin(name) {
    const confirmed = await OdinUI.confirm(
      `Are you sure you want to delete the plugin "${name}"? This action cannot be undone.`,
      'Delete Plugin'
    );

    if (!confirmed) return;

    try {
      await OdinAPI.plugins.delete(name);
      OdinUI.showToast(`Plugin "${name}" deleted successfully`, 'success');
      setTimeout(() => {
        window.location.href = '/admin/plugins';
      }, 500);
    } catch (error) {
      OdinUI.showToast(`Failed to delete plugin: ${error.message}`, 'danger');
    }
  }

  /**
   * Test plugin
   */
  async function testPlugin(name) {
    const testData = {
      method: 'GET',
      path: '/test',
      headers: {
        'Content-Type': ['application/json'],
      },
      body: '',
    };

    try {
      OdinUI.showLoading('#test-results', 'Testing plugin...');
      const results = await OdinAPI.plugins.test(name, testData);

      const resultsHtml = generateTestResultsHtml(results);
      document.getElementById('test-results').innerHTML = resultsHtml;
      OdinUI.showToast('Plugin test completed', 'success');
    } catch (error) {
      OdinUI.showError('#test-results', `Test failed: ${error.message}`);
    }
  }

  /**
   * Generate test results HTML
   */
  function generateTestResultsHtml(results) {
    let html = '<div class="card"><div class="card-body">';
    html += '<h5 class="card-title">Test Results</h5>';

    if (results.results) {
      html += '<table class="table table-sm">';
      html += '<thead><tr><th>Hook</th><th>Status</th><th>Error</th></tr></thead>';
      html += '<tbody>';

      for (const [hook, result] of Object.entries(results.results)) {
        const statusClass = result.success ? 'text-success' : 'text-danger';
        const statusIcon = result.success ? '✓' : '✗';
        html += `<tr>
          <td>${OdinUI.escapeHtml(hook)}</td>
          <td class="${statusClass}">${statusIcon} ${
          result.success ? 'Success' : 'Failed'
        }</td>
          <td>${result.error ? OdinUI.escapeHtml(result.error) : '-'}</td>
        </tr>`;
      }

      html += '</tbody></table>';
    }

    if (results.context) {
      html += '<details class="mt-3">';
      html += '<summary>Context Details</summary>';
      html +=
        '<pre class="mt-2 p-2 bg-light rounded">' +
        OdinUI.escapeHtml(JSON.stringify(results.context, null, 2)) +
        '</pre>';
      html += '</details>';
    }

    html += '</div></div>';
    return html;
  }

  /**
   * Initialize plugin detail page
   */
  function initPluginDetail() {
    const editBtn = document.getElementById('edit-plugin-btn');
    const saveBtn = document.getElementById('save-plugin-btn');
    const cancelBtn = document.getElementById('cancel-edit-btn');

    if (editBtn) {
      editBtn.addEventListener('click', toggleEditMode);
    }

    if (saveBtn) {
      saveBtn.addEventListener('click', savePluginChanges);
    }

    if (cancelBtn) {
      cancelBtn.addEventListener('click', () => toggleEditMode());
    }
  }

  /**
   * Toggle edit mode on plugin detail page
   */
  function toggleEditMode() {
    const viewMode = document.getElementById('viewMode');
    const editMode = document.getElementById('editMode');

    if (viewMode && editMode) {
      if (viewMode.style.display === 'none') {
        viewMode.style.display = 'block';
        editMode.style.display = 'none';
      } else {
        viewMode.style.display = 'none';
        editMode.style.display = 'block';
      }
    }
  }

  /**
   * Save plugin changes
   */
  async function savePluginChanges() {
    const form = document.getElementById('edit-plugin-form');
    if (!form) return;

    const formData = new FormData(form);
    const name = form.dataset.pluginName;

    const data = {
      version: formData.get('version'),
      description: formData.get('description') || '',
      enabled: formData.get('enabled') === 'on',
      config: parseConfig(formData.get('config')),
      hooks: formData.getAll('hooks'),
    };

    try {
      await OdinAPI.plugins.update(name, data);
      OdinUI.showToast('Plugin updated successfully', 'success');
      setTimeout(() => location.reload(), 500);
    } catch (error) {
      OdinUI.showToast(`Failed to update plugin: ${error.message}`, 'danger');
    }
  }

  // Auto-initialize on DOMContentLoaded
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      initPluginForm();
      initPluginDetail();
    });
  } else {
    initPluginForm();
    initPluginDetail();
  }

  return {
    initPluginForm,
    initPluginDetail,
    togglePlugin,
    deletePlugin,
    testPlugin,
    parseConfig,
    TEMPLATES,
  };
})();

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
  module.exports = OdinPlugins;
}
