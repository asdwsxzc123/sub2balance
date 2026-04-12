// API Base URL
const API_BASE = '/api';

// API Helper Functions
const api = {
  // Make authenticated request
  async request(method, endpoint, data = null) {
    const token = localStorage.getItem('token');
    const headers = {
      'Content-Type': 'application/json',
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const options = {
      method,
      headers,
    };

    if (data) {
      options.body = JSON.stringify(data);
    }

    try {
      const response = await fetch(`${API_BASE}${endpoint}`, options);
      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.error || `HTTP ${response.status}`);
      }

      return result;
    } catch (error) {
      console.error('API Error:', error);
      throw error;
    }
  },

  get(endpoint) {
    return this.request('GET', endpoint);
  },

  post(endpoint, data) {
    return this.request('POST', endpoint, data);
  },

  put(endpoint, data) {
    return this.request('PUT', endpoint, data);
  },

  delete(endpoint) {
    return this.request('DELETE', endpoint);
  },
};

// Auth Functions
const auth = {
  async login(email, password) {
    const response = await api.post('/auth/login', { email, password });
    localStorage.setItem('token', response.token);
    localStorage.setItem('user', JSON.stringify(response.user));
    return response;
  },

  async logout() {
    try {
      await api.post('/auth/logout');
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login.html';
    }
  },

  async me() {
    return await api.get('/auth/me');
  },

  getUser() {
    const userStr = localStorage.getItem('user');
    return userStr ? JSON.parse(userStr) : null;
  },

  getToken() {
    return localStorage.getItem('token');
  },

  isAuthenticated() {
    return !!this.getToken();
  },

  isAdmin() {
    const user = this.getUser();
    return user && user.role === 'admin';
  },

  requireAuth() {
    if (!this.isAuthenticated()) {
      window.location.href = '/login.html';
      return false;
    }
    return true;
  },

  requireAdmin() {
    if (!this.requireAuth()) return false;
    if (!this.isAdmin()) {
      alert('Admin access required');
      window.location.href = '/staff.html';
      return false;
    }
    return true;
  },
};

// Utility Functions
const utils = {
  formatDate(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleString();
  },

  formatStatus(status) {
    const statusMap = {
      pending: 'Pending',
      approved: 'Approved',
      rejected: 'Rejected',
      completed: 'Completed',
      failed: 'Failed',
    };
    return statusMap[status] || status;
  },

  getStatusClass(status) {
    const classMap = {
      pending: 'bg-yellow-100 text-yellow-800',
      approved: 'bg-blue-100 text-blue-800',
      rejected: 'bg-red-100 text-red-800',
      completed: 'bg-green-100 text-green-800',
      failed: 'bg-red-100 text-red-800',
    };
    return classMap[status] || 'bg-gray-100 text-gray-800';
  },

  showError(message) {
    alert(`Error: ${message}`);
  },

  showSuccess(message) {
    alert(message);
  },
};

// Export to global scope
window.api = api;
window.auth = auth;
window.utils = utils;
