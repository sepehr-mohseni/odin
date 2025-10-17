/**
 * Odin API Gateway - API Module
 * Centralized API communication functions
 */

const OdinAPI = (() => {
  'use strict';

  const baseURL = '/admin/api';

  /**
   * Generic fetch wrapper with error handling
   */
  async function request(url, options = {}) {
    const defaultOptions = {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    };

    const config = { ...defaultOptions, ...options };

    try {
      const response = await fetch(url, config);

      // Handle redirects
      const redirectTo = response.headers.get('HX-Redirect');
      if (redirectTo) {
        window.location.href = redirectTo;
        return null;
      }

      // Handle unauthorized
      if (response.status === 401 || response.status === 403) {
        window.location.href = '/admin/login';
        return null;
      }

      // Parse response
      const contentType = response.headers.get('content-type');
      let data;
      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else {
        data = await response.text();
      }

      if (!response.ok) {
        throw new Error(data.error || data || `HTTP error! status: ${response.status}`);
      }

      return data;
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // Plugin API
  const plugins = {
    list: () => request(`${baseURL}/plugins`),
    
    get: (name) => request(`${baseURL}/plugins/${name}`),
    
    create: (data) =>
      request(`${baseURL}/plugins`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    update: (name, data) =>
      request(`${baseURL}/plugins/${name}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    
    delete: (name) =>
      request(`${baseURL}/plugins/${name}`, {
        method: 'DELETE',
      }),
    
    enable: (name) =>
      request(`${baseURL}/plugins/${name}/enable`, {
        method: 'POST',
      }),
    
    disable: (name) =>
      request(`${baseURL}/plugins/${name}/disable`, {
        method: 'POST',
      }),
    
    load: (name) =>
      request(`${baseURL}/plugins/${name}/load`, {
        method: 'POST',
      }),
    
    unload: (name) =>
      request(`${baseURL}/plugins/${name}/unload`, {
        method: 'POST',
      }),
    
    test: (name, testData) =>
      request(`${baseURL}/plugins/test/${name}`, {
        method: 'POST',
        body: JSON.stringify(testData),
      }),
    
    upload: async (formData) => {
      // Don't set Content-Type header - let browser set it with boundary
      const response = await fetch(`${baseURL}/plugins/upload`, {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Upload failed');
      }

      return response.json();
    },
    
    build: (data) =>
      request(`${baseURL}/plugins/build`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  };

  // Service API
  const services = {
    list: () => request(`${baseURL}/services`),
    
    get: (name) => request(`${baseURL}/services/${name}`),
    
    create: (data) =>
      request(`${baseURL}/services`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    update: (name, data) =>
      request(`${baseURL}/services/${name}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    
    delete: (name) =>
      request(`${baseURL}/services/${name}`, {
        method: 'DELETE',
      }),
  };

  // Postman Integration API
  const postman = {
    listCollections: (apiKey) =>
      request(`/admin/api/postman/collections?apiKey=${encodeURIComponent(apiKey)}`),
    
    importCollection: (collectionId, apiKey, serviceName = null) =>
      request(`/admin/api/postman/collections/${collectionId}/import`, {
        method: 'POST',
        body: JSON.stringify({ apiKey, serviceName }),
      }),
    
    exportCollection: (collectionId, apiKey) =>
      request(`/admin/api/postman/collections/${collectionId}/export`, {
        method: 'POST',
        body: JSON.stringify({ apiKey }),
      }),
    
    triggerHook: (hookType, hookData) =>
      request(`/admin/api/postman/hooks/${hookType}`, {
        method: 'POST',
        body: JSON.stringify(hookData),
      }),
  };

  // Health/Monitoring API
  const monitoring = {
    health: () => request('/health'),
    ready: () => request('/ready'),
    metrics: () => fetch('/metrics').then((r) => r.text()),
  };

  return {
    request,
    plugins,
    services,
    postman,
    monitoring,
  };
})();

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
  module.exports = OdinAPI;
}
