/**
 * Odin API Gateway - UI Utilities Module
 * Common UI helper functions for notifications, modals, etc.
 */

const OdinUI = (() => {
  'use strict';

  /**
   * Show a toast notification
   */
  function showToast(message, type = 'info', duration = 3000) {
    // Create toast container if it doesn't exist
    let container = document.getElementById('toast-container');
    if (!container) {
      container = document.createElement('div');
      container.id = 'toast-container';
      container.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        z-index: 9999;
        max-width: 350px;
      `;
      document.body.appendChild(container);
    }

    // Create toast element
    const toast = document.createElement('div');
    toast.className = `alert alert-${type} alert-dismissible fade show`;
    toast.style.cssText = 'margin-bottom: 10px; box-shadow: 0 4px 12px rgba(0,0,0,0.15);';
    toast.setAttribute('role', 'alert');

    const icon = getIconForType(type);
    toast.innerHTML = `
      ${icon} ${escapeHtml(message)}
      <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    `;

    container.appendChild(toast);

    // Auto-remove after duration
    setTimeout(() => {
      toast.classList.remove('show');
      setTimeout(() => toast.remove(), 150);
    }, duration);

    return toast;
  }

  /**
   * Get icon for toast type
   */
  function getIconForType(type) {
    const icons = {
      success: '<i class="bi bi-check-circle-fill"></i>',
      danger: '<i class="bi bi-exclamation-triangle-fill"></i>',
      warning: '<i class="bi bi-exclamation-circle-fill"></i>',
      info: '<i class="bi bi-info-circle-fill"></i>',
    };
    return icons[type] || icons.info;
  }

  /**
   * Show loading spinner
   */
  function showLoading(element, message = 'Loading...') {
    if (typeof element === 'string') {
      element = document.querySelector(element);
    }
    if (!element) return;

    element.innerHTML = `
      <div class="text-center py-5">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">${escapeHtml(message)}</span>
        </div>
        <p class="mt-3 text-muted">${escapeHtml(message)}</p>
      </div>
    `;
  }

  /**
   * Hide loading spinner and show content
   */
  function hideLoading(element, content = '') {
    if (typeof element === 'string') {
      element = document.querySelector(element);
    }
    if (!element) return;

    element.innerHTML = content;
  }

  /**
   * Show error message
   */
  function showError(element, message, details = null) {
    if (typeof element === 'string') {
      element = document.querySelector(element);
    }
    if (!element) return;

    const detailsHtml = details
      ? `<details class="mt-2"><summary>Details</summary><pre class="mt-2 p-2 bg-light rounded">${escapeHtml(
          JSON.stringify(details, null, 2)
        )}</pre></details>`
      : '';

    element.innerHTML = `
      <div class="alert alert-danger" role="alert">
        <i class="bi bi-exclamation-triangle-fill me-2"></i>
        <strong>Error:</strong> ${escapeHtml(message)}
        ${detailsHtml}
      </div>
    `;
  }

  /**
   * Confirm dialog
   */
  function confirm(message, title = 'Confirm') {
    return new Promise((resolve) => {
      const modal = createModal(title, message, [
        {
          text: 'Cancel',
          class: 'btn-secondary',
          onClick: () => {
            modal.hide();
            resolve(false);
          },
        },
        {
          text: 'Confirm',
          class: 'btn-primary',
          onClick: () => {
            modal.hide();
            resolve(true);
          },
        },
      ]);
      modal.show();
    });
  }

  /**
   * Create a modal dialog
   */
  function createModal(title, body, buttons = []) {
    const modalId = 'modal-' + Date.now();
    const buttonsHtml = buttons
      .map(
        (btn) =>
          `<button type="button" class="btn ${btn.class || 'btn-secondary'}" data-action="${btn.text}">${escapeHtml(
            btn.text
          )}</button>`
      )
      .join('');

    const modalHtml = `
      <div class="modal fade" id="${modalId}" tabindex="-1">
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title">${escapeHtml(title)}</h5>
              <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
              ${typeof body === 'string' ? escapeHtml(body) : body}
            </div>
            <div class="modal-footer">
              ${buttonsHtml}
            </div>
          </div>
        </div>
      </div>
    `;

    const modalEl = document.createElement('div');
    modalEl.innerHTML = modalHtml;
    document.body.appendChild(modalEl.firstElementChild);

    const modal = new bootstrap.Modal(document.getElementById(modalId));

    // Attach button event listeners
    buttons.forEach((btn) => {
      const buttonEl = document.querySelector(
        `#${modalId} button[data-action="${btn.text}"]`
      );
      if (buttonEl && btn.onClick) {
        buttonEl.addEventListener('click', btn.onClick);
      }
    });

    // Clean up on hide
    document.getElementById(modalId).addEventListener('hidden.bs.modal', () => {
      document.getElementById(modalId).remove();
    });

    return modal;
  }

  /**
   * Escape HTML to prevent XSS
   */
  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  /**
   * Format date
   */
  function formatDate(date, format = 'datetime') {
    const d = new Date(date);
    if (isNaN(d)) return 'Invalid date';

    const pad = (n) => n.toString().padStart(2, '0');

    if (format === 'date') {
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
    } else if (format === 'time') {
      return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
    } else if (format === 'datetime') {
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(
        d.getMinutes()
      )}:${pad(d.getSeconds())}`;
    } else if (format === 'relative') {
      return formatRelativeTime(d);
    }
    return d.toLocaleString();
  }

  /**
   * Format relative time (e.g., "2 hours ago")
   */
  function formatRelativeTime(date) {
    const now = new Date();
    const diff = now - new Date(date);
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days} day${days > 1 ? 's' : ''} ago`;
    if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''} ago`;
    if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
    return `${seconds} second${seconds !== 1 ? 's' : ''} ago`;
  }

  /**
   * Format bytes to human readable
   */
  function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i];
  }

  /**
   * Copy text to clipboard
   */
  async function copyToClipboard(text) {
    try {
      await navigator.clipboard.writeText(text);
      showToast('Copied to clipboard!', 'success', 2000);
      return true;
    } catch (err) {
      console.error('Failed to copy:', err);
      showToast('Failed to copy to clipboard', 'danger');
      return false;
    }
  }

  /**
   * Debounce function
   */
  function debounce(func, wait = 300) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }

  /**
   * Throttle function
   */
  function throttle(func, limit = 300) {
    let inThrottle;
    return function executedFunction(...args) {
      if (!inThrottle) {
        func(...args);
        inThrottle = true;
        setTimeout(() => (inThrottle = false), limit);
      }
    };
  }

  /**
   * Set active nav link based on current path
   */
  function setActiveNavLink() {
    const path = window.location.pathname;
    const navLinks = document.querySelectorAll('.nav-link');
    navLinks.forEach((link) => {
      link.classList.remove('active');
      if (link.getAttribute('href') === path) {
        link.classList.add('active');
      }
    });
  }

  /**
   * Initialize on DOMContentLoaded
   */
  function init() {
    setActiveNavLink();

    // Setup HTMX event handlers if HTMX is available
    if (typeof htmx !== 'undefined') {
      document.addEventListener('htmx:afterSwap', (event) => {
        const redirectTo = event.detail.xhr.getResponseHeader('HX-Redirect');
        if (redirectTo) {
          window.location.href = redirectTo;
        }
      });

      document.addEventListener('htmx:responseError', (event) => {
        if (event.detail.xhr.status === 401 || event.detail.xhr.status === 403) {
          window.location.href = '/admin/login';
        }
      });
    }
  }

  // Auto-initialize on DOMContentLoaded
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  return {
    showToast,
    showLoading,
    hideLoading,
    showError,
    confirm,
    createModal,
    escapeHtml,
    formatDate,
    formatRelativeTime,
    formatBytes,
    copyToClipboard,
    debounce,
    throttle,
    setActiveNavLink,
    init,
  };
})();

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
  module.exports = OdinUI;
}
