let currentUser = null;
let wsSocket = null;
let wsReconnectTimer = null;
let dashboardStatsInflight = null;
let dashboardStatsRefreshTimer = null;
let managedDevices = [];
let discoveredDevices = [];

// Global Pagination State
const PAGINATION_SIZE = 20;
let employeesPage = 1;
let departmentsPage = 1;
let positionsPage = 1;
let travelPage = 1;
let leavesPage = 1;

/**
 * Generic Pagination Renderer
 * @param {string} containerId - The ID of the pagination container
 * @param {number} totalItems - Total number of items
 * @param {number} currentPage - Current active page
 * @param {string} callbackName - Global function name to call on page change (e.g. 'changeEmployeesPage')
 * @param {string} countId - ID of the element to show "Showing X-Y of Z"
 */
function renderCommonPagination(containerId, totalItems, currentPage, callbackName, countId) {
    const el = document.getElementById(containerId);
    const countEl = document.getElementById(countId);
    if (!el) return;

    const totalPages = Math.ceil(totalItems / PAGINATION_SIZE);
    
    // Update count display
    if (countEl) {
        const start = totalItems === 0 ? 0 : (currentPage - 1) * PAGINATION_SIZE + 1;
        const end = Math.min(currentPage * PAGINATION_SIZE, totalItems);
        countEl.textContent = `Mostrando ${start}-${end} de ${totalItems} registro${totalItems !== 1 ? 's' : ''}`;
    }

    if (totalPages <= 1) {
        el.innerHTML = '';
        return;
    }

    let html = `<button class="btn btn-secondary btn-sm" ${currentPage === 1 ? 'disabled' : ''} onclick="${callbackName}(${currentPage - 1})">Anterior</button>`;
    
    const startPage = Math.max(1, currentPage - 2);
    const endPage = Math.min(totalPages, startPage + 4);
    
    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="btn btn-sm ${i === currentPage ? 'btn-primary' : 'btn-secondary'}" onclick="${callbackName}(${i})">${i}</button>`;
    }

    html += `<button class="btn btn-secondary btn-sm" ${currentPage === totalPages ? 'disabled' : ''} onclick="${callbackName}(${currentPage + 1})">Siguiente</button>`;
    el.innerHTML = html;
}

function escapeHTML(value) {
    return String(value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function encodeInlineArg(value) {
    return encodeURIComponent(String(value ?? ''));
}

function decodeInlineArg(value) {
    return decodeURIComponent(String(value ?? ''));
}

function initTheme() {
    const btn = document.getElementById('btn-theme-toggle');
    const body = document.body;
    
    // Load preference
    const currentTheme = localStorage.getItem('theme') || 'dark';
    if (currentTheme === 'light') {
        body.setAttribute('data-theme', 'light');
    } else {
        body.removeAttribute('data-theme');
    }
    
    if (btn) {
        btn.addEventListener('click', () => {
            const isLight = body.getAttribute('data-theme') === 'light';
            if (isLight) {
                body.removeAttribute('data-theme');
                localStorage.setItem('theme', 'dark');
            } else {
                body.setAttribute('data-theme', 'light');
                localStorage.setItem('theme', 'light');
            }
        });
    }
}

document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    initLeavesUI();
    initSystemIcons();
    initDashboardUI();
    validateSession();

    initLogin();
    initLogout();
    initNavigation();
    initMobileMenu();
    initSupportUI();
    initScan();
    initEmployeeUI();
    initReports();
    initFaceUI();
    initConfig();
    initLDAP();
    initUsers();
    initTabs();
    initDepartments();
    initPositions();
    initDeviceManager();
    initDeviceErrorManager();
    initTravelAllowances();
    initPWA();
    initAttendanceReport();
});

function initSystemIcons() {
    const iconMap = {
        settings: { id: 'icon-settings', title: 'Configuración' }
    };

    Object.entries(iconMap).forEach(([page, meta]) => {
        const item = document.querySelector(`.sidebar nav li[data-page="${page}"]`);
        if (!item || item.querySelector('.nav-item-content')) return;

        item.dataset.title = meta.title;
        item.innerHTML = `<span class="nav-item-content"><svg class="nav-icon" viewBox="0 0 24 24" aria-hidden="true"><use href="#${meta.id}"></use></svg><span class="nav-text">${meta.title}</span></span>`;
    });
}

function initDashboardUI() {
    const statCards = [
        { id: 'stats-late', icon: 'icon-alert', helper: 'Casos fuera de tolerancia', tone: 'negative' },
        { id: 'stats-absent', icon: 'icon-attendance', helper: 'Sin marcacion en la jornada actual', tone: '' },
        { id: 'stats-devices', icon: 'icon-devices', helper: 'Sin terminal predeterminada', tone: '', helperId: 'dashboard-device-state' }
    ];

    statCards.forEach(({ id, icon, helper, tone, helperId }) => {
        const number = document.getElementById(id);
        if (!number) return;

        const card = number.closest('.stat-card');
        if (!card) return;

        if (!card.querySelector('.stat-card-header')) {
            const heading = card.querySelector('h3');
            if (heading) {
                const header = document.createElement('div');
                header.className = 'stat-card-header';
                header.innerHTML = `<span class="stat-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><use href="#${icon}"></use></svg></span>`;
                heading.parentNode.insertBefore(header, heading);
                header.appendChild(heading);
            }
        }

        let helperEl = card.querySelector('.trend');
        if (!helperEl) {
            helperEl = document.createElement('span');
            helperEl.className = 'trend';
            card.appendChild(helperEl);
        }

        helperEl.textContent = helper;
        helperEl.className = `trend${tone ? ` ${tone}` : ''}`;
        if (helperId) {
            helperEl.id = helperId;
        }
    });
}

function initSupportUI() {
    const wrap = document.getElementById('support-fab-wrap');
    const button = document.getElementById('support-fab-button');
    const panel = document.getElementById('support-fab-panel');

    if (!wrap || !button || !panel) return;

    button.addEventListener('click', (event) => {
        event.stopPropagation();
        panel.classList.toggle('active');
    });

    panel.addEventListener('click', (event) => {
        event.stopPropagation();
    });

    document.addEventListener('click', (event) => {
        if (!wrap.contains(event.target) && !panel.contains(event.target)) {
            panel.classList.remove('active');
        }
    });

    document.addEventListener('keydown', (event) => {
        if (event.key === 'Escape') {
            panel.classList.remove('active');
        }
    });
}

async function loadDashboardStats() {
    if (dashboardStatsInflight) {
        return dashboardStatsInflight;
    }

    dashboardStatsInflight = (async () => {
    try {
        const resp = await fetch('/api/attendance/stats', {
            headers: getAuthHeaders()
        });
        if (!resp.ok) return;

        const stats = await resp.json();
        const mappings = {
            'stats-present': stats.present ?? 0,
            'stats-late': stats.late ?? 0,
            'stats-absent': stats.absent ?? 0,
            'stats-devices': stats.devices ?? 0
        };

        Object.entries(mappings).forEach(([id, value]) => {
            const el = document.getElementById(id);
            if (el) {
                el.innerText = value;
            }
        });

        // Update dashboard events table
        const tbody = document.querySelector('#dashboard-events-table tbody');
        if (tbody && stats.recentEvents) {
            if (stats.recentEvents.length === 0) {
                tbody.innerHTML = '<tr><td colspan="3" class="text-muted" style="text-align: center; padding: 2rem;">No hay actividad reciente.</td></tr>';
            } else {
                tbody.innerHTML = stats.recentEvents.map(ev => `
                    <tr>
                        <td>${new Date(ev.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</td>
                        <td>
                            <div style="display:flex; flex-direction:column;">
                                <strong>${escapeHTML(ev.employeeName)}</strong>
                                <span style="font-size:0.7rem; color:var(--text-muted);">ID: ${ev.employeeNo}</span>
                            </div>
                        </td>
                        <td><span class="badge badge-secondary" style="font-size:0.65rem;">${escapeHTML(ev.deviceId)}</span></td>
                    </tr>
                `).join('');
            }
        }

        const dateLabel = document.getElementById('dashboard-date-label');
        if (dateLabel) {
            const rawDate = stats.date ? new Date(`${stats.date}T00:00:00`) : new Date();
            dateLabel.innerText = rawDate.toLocaleDateString('es-ES', {
                day: '2-digit',
                month: 'long',
                year: 'numeric'
            });
        }
    } catch (err) {
        console.error('Dashboard stats failed', err);
    } finally {
        dashboardStatsInflight = null;
    }
    })();

    return dashboardStatsInflight;
}

function scheduleDashboardStatsRefresh(delay = 250) {
    if (dashboardStatsRefreshTimer) {
        return;
    }
    dashboardStatsRefreshTimer = setTimeout(() => {
        dashboardStatsRefreshTimer = null;
        loadDashboardStats();
    }, delay);
}

// ==================== AUTHENTICATION ====================

function showLoginScreen() {
    document.getElementById('login-screen').style.display = 'flex';
    document.getElementById('app-container').style.display = 'none';
}

function hideLoginScreen() {
    document.getElementById('login-screen').style.display = 'none';
    document.getElementById('app-container').style.display = 'flex';
}

async function validateSession() {
    try {
        const token = sessionStorage.getItem('token');
        if (!token) {
            logout();
            return;
        }

        const resp = await fetch('/api/auth/me', {
            headers: getAuthHeaders()
        });

        if (resp.ok) {
            currentUser = await resp.json();
            // Ensure token is attached back if me response doesn't include it
            if (!currentUser.token) currentUser.token = token;
            
            hideLoginScreen();
            updateUserInfo();
            applyRolePermissions();
            initWebSocket();
            loadManagedDevices();
            loadDashboardStats();
        } else {
            logout();
        }
    } catch (err) {
        logout();
    }
}

function initLogin() {
    const form = document.getElementById('login-form');
    if (!form) return;

    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;
        const errorEl = document.getElementById('login-error');
        const submitBtn = form.querySelector('button[type="submit"]');

        errorEl.textContent = '';
        submitBtn.disabled = true;
        submitBtn.textContent = 'Autenticando...';

        try {
            const resp = await fetch('/api/public/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });

            const data = await resp.json();

            if (resp.ok) {
                currentUser = data.user;
                if (data.token) {
                    currentUser.token = data.token;
                    sessionStorage.setItem('token', data.token);
                }

                hideLoginScreen();
                updateUserInfo();
                applyRolePermissions();
                loadConfig();
                initWebSocket();
                loadManagedDevices();
                loadDashboardStats();
                showToast('¡Bienvenido, ' + (currentUser.fullName || currentUser.username) + '!');
            } else {
                errorEl.textContent = data.error || 'Credenciales inválidas';
            }
        } catch (err) {
            errorEl.textContent = 'Error de conexión. Verifica que el servidor esté ejecutándose.';
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Iniciar Sesión';
        }
    });
}

function initLogout() {
    const btnLogout = document.getElementById('btn-logout');
    const logoutModal = document.getElementById('logout-modal');
    const confirmBtn = document.getElementById('confirm-logout');
    const closeBtns = logoutModal.querySelectorAll('.close-modal');

    if (btnLogout) {
        btnLogout.addEventListener('click', () => {
            logoutModal.classList.add('active');
        });
    }

    closeBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            logoutModal.classList.remove('active');
        });
    });

    if (confirmBtn) {
        confirmBtn.addEventListener('click', () => {
            logout();
            logoutModal.classList.remove('active');
        });
    }
}

async function logout() {
    closeMobileMenu();
    const supportWrap = document.getElementById('support-fab-wrap');
    if (supportWrap) {
        supportWrap.classList.remove('open');
    }
    sessionStorage.removeItem('token');
    currentUser = null;
    if (wsReconnectTimer) {
        clearTimeout(wsReconnectTimer);
        wsReconnectTimer = null;
    }
    if (wsSocket) {
        wsSocket.close();
        wsSocket = null;
    }
    try {
        await fetch('/api/auth/logout', {
            method: 'POST',
            headers: getAuthHeaders()
        });
    } catch (err) {}
    applyRolePermissions();
    showLoginScreen();

    // Reset form
    document.getElementById('login-form').reset();
    document.getElementById('login-error').textContent = '';
}

function updateUserInfo() {
    if (currentUser) {
        const usernameEl = document.getElementById('current-username');
        if (usernameEl) {
            usernameEl.textContent = currentUser.fullName || currentUser.username;
        }
    }
}

function getCurrentRole() {
    return currentUser?.role || 'viewer';
}

function hasAnyRole(...roles) {
    return roles.includes(getCurrentRole());
}

function isAdmin() {
    return hasAnyRole('admin');
}

function canManageDevices() {
    return isAdmin();
}

function canManageOrganization() {
    return hasAnyRole('admin', 'manager');
}

function canManageLeaves() {
    return hasAnyRole('admin', 'manager');
}

function canManageTravel() {
    return hasAnyRole('admin', 'manager');
}

function canManageTravelRates() {
    return isAdmin();
}

function canManageUsers() {
    return isAdmin();
}

function applyRolePermissions() {
    const devicesNav = document.querySelector('.sidebar nav li[data-page="devices"]');
    const settingsNav = document.querySelector('.sidebar nav li[data-page="settings"]');
    const settingsPage = document.getElementById('settings');
    const currentActivePage = document.querySelector('.page.active');

    if (devicesNav) {
        devicesNav.classList.toggle('is-hidden-by-role', !canManageDevices());
    }
    toggleRoleElement('btn-add-device', canManageDevices());
    toggleRoleElement('btn-add-device-inline', canManageDevices());
    toggleRoleElement('btn-refresh-devices', canManageDevices());
    toggleRoleElement('btn-scan-devices-inline', canManageDevices());
    toggleRoleElement('btn-add-employee', canManageOrganization());
    toggleRoleElement('btn-sync-employees', canManageDevices());
    toggleRoleElement('btn-add-dept', canManageOrganization());
    toggleRoleElement('btn-add-pos', canManageOrganization());
    toggleRoleElement('btn-new-leave', canManageLeaves());
    toggleRoleElement('btn-new-travel', canManageTravel());
    toggleRoleElement('btn-manage-rates', canManageTravelRates());

    if (settingsNav) {
        settingsNav.classList.toggle('is-hidden-by-role', !isAdmin());
    }
    if (settingsPage) {
        settingsPage.classList.toggle('is-hidden-by-role', !isAdmin());
    }

    // Toggle Travel Module visibility
    const travelNav = document.querySelector('.sidebar nav li[data-page="travel-allowances"]');
    const travelRatesNav = document.querySelector('.sidebar nav li[data-page="travel-rates"]');
    const isTravelEnabled = window.isTravelModuleEnabled !== false; // Default to true if not set

    if (travelNav) {
        travelNav.classList.toggle('is-hidden-by-feature', !isTravelEnabled);
        if (!isTravelEnabled && travelNav.classList.contains('active')) {
             document.querySelector('.sidebar nav li[data-page="dashboard"]')?.click();
        }
    }
    if (travelRatesNav) {
        travelRatesNav.classList.toggle('is-hidden-by-feature', !isTravelEnabled);
    }

    if (currentActivePage && ((!isAdmin() && currentActivePage.id === 'settings') || (!canManageDevices() && currentActivePage.id === 'devices'))) {
        document.querySelector('.sidebar nav li[data-page="dashboard"]')?.click();
    }
}

function toggleRoleElement(id, visible) {
    const el = document.getElementById(id);
    if (!el) return;
    el.classList.toggle('is-hidden-by-role', !visible);
}

function getAuthHeaders() {
	const token = currentUser?.token || sessionStorage.getItem('token') || localStorage.getItem('auth_token');
	return {
		'Content-Type': 'application/json',
		...(token ? { 'Authorization': `Bearer ${token}` } : {})
	};
}

window.currentEmployeePhoto = null;
window.currentEditingEmployee = null;
window.employeePhotoRemoved = false;

function isSystemAdminEmployee(employee) {
	return Boolean(employee && employee.isSystemAdmin);
}

function updateEmployeeAdminUI(employee) {
	const removeBtn = document.getElementById('btn-remove-employee-photo');
	const adminFlag = document.getElementById('employee-admin-flag');
	const isAdminEmployee = isSystemAdminEmployee(employee);

	if (removeBtn) {
		removeBtn.disabled = isAdminEmployee;
		removeBtn.title = isAdminEmployee ? 'La foto de un administrador del sistema no se puede quitar' : '';
		removeBtn.style.opacity = isAdminEmployee ? '0.65' : '';
		removeBtn.style.cursor = isAdminEmployee ? 'not-allowed' : '';
	}

	if (adminFlag) {
		adminFlag.style.display = isAdminEmployee ? 'block' : 'none';
	}
}

function employeePhotoSrc(employee) {
	if (!employee) return '';
	if (employee.photoData) {
		const mime = String(employee.photoData).startsWith('iVBOR') ? 'image/png' : 'image/jpeg';
		return `data:${mime};base64,${employee.photoData}`;
	}
	return employee.photoUrl || '';
}

function setEmployeePhotoState(src, placeholderText = '') {
	const photoPreview = document.getElementById('employee-photo-preview');
	const photoPlaceholder = document.getElementById('employee-photo-placeholder');
	const removeBtn = document.getElementById('btn-remove-employee-photo');
	if (!photoPreview || !photoPlaceholder) return;

	if (src) {
		photoPreview.src = src;
		photoPreview.style.display = 'block';
		photoPlaceholder.style.display = 'none';
		if (removeBtn) removeBtn.style.display = 'inline-flex';
		return;
	}

	photoPreview.src = '';
	photoPreview.style.display = 'none';
	photoPlaceholder.style.display = 'flex';
	photoPlaceholder.innerHTML = `<svg viewBox="0 0 24 24" style="width: 54px; height: 54px; opacity: 0.2;"><use href="#icon-user"></use></svg>${placeholderText ? `<span style="font-size: 0.65rem; color: var(--text-muted); margin-top: 5px; text-align: center;">${escapeHTML(placeholderText)}</span>` : ''}`;
	if (removeBtn) removeBtn.style.display = 'none';
}

window.handlePhotoSelect = function(event) {
	const input = event?.target || event;
	const file = input?.files?.[0];
	if (!file) return;

	if (!['image/jpeg', 'image/png'].includes(file.type)) {
		showToast('Solo se permiten imagenes JPG o PNG', 'error');
		input.value = '';
		return;
	}
	if (file.size > 5 * 1024 * 1024) {
		showToast('La foto no puede superar 5MB', 'error');
		input.value = '';
		return;
	}

	const reader = new FileReader();
	reader.onload = (re) => {
		const dataUrl = String(re.target.result || '');
		window.currentEmployeePhoto = dataUrl.includes(',') ? dataUrl.split(',')[1] : dataUrl;
		window.employeePhotoRemoved = false;
		const photoUrlInput = document.getElementById('employee-photo-url');
		if (photoUrlInput) photoUrlInput.value = '';
		setEmployeePhotoState(dataUrl);
	};
	reader.readAsDataURL(file);
};

window.removeEmployeePhoto = async function() {
	const employee = window.currentEditingEmployee;
	if (isSystemAdminEmployee(employee)) {
		showToast('La foto de este administrador del sistema esta protegida', 'warning');
		return;
	}
	const hadStoredPhoto = Boolean(employeePhotoSrc(employee));
	const employeeNo = employee?.employeeNo || '';

	window.currentEmployeePhoto = null;
	window.employeePhotoRemoved = hadStoredPhoto;

	const photoInput = document.getElementById('input-employee-photo');
	const photoUrlInput = document.getElementById('employee-photo-url');
	if (photoInput) photoInput.value = '';
	if (photoUrlInput) photoUrlInput.value = '';

	setEmployeePhotoState('', 'Sin foto local');

	if (!hadStoredPhoto || !employeeNo) {
		return;
	}

	const removed = await deleteEmployeePhoto(employeeNo, { silent: true });
	if (!removed) {
		if (employee) {
			setEmployeePhotoState(employeePhotoSrc(employee), 'Sin foto local');
		}
		return;
	}

	window.employeePhotoRemoved = false;
	showToast('Foto eliminada');
};

window.uploadEmployeePhoto = async function(employeeNo, options = {}) {
	if (!window.currentEmployeePhoto) return true;
	if (!employeeNo) {
		showToast('No se encontro el numero de empleado para subir la foto', 'error');
		return false;
	}

	try {
		const resp = await fetch(`/api/employees/${encodeURIComponent(employeeNo)}/photo`, {
			method: 'PUT',
			headers: getAuthHeaders(),
			body: JSON.stringify({ photo: window.currentEmployeePhoto })
		});
		const result = await safeReadJSON(resp);

		if (!resp.ok) {
			showToast(result.error || 'No se pudo guardar la foto', 'error');
			return false;
		}

		window.currentEmployeePhoto = null;
		const input = document.getElementById('input-employee-photo');
		if (input) input.value = '';

		if (!options.silent) {
			const devicesFailed = result?.sync?.devicesFailed || 0;
			const devicesSuccess = result?.sync?.devicesSuccess || 0;
			const devicesTotal = result?.sync?.devicesTotal || 0;
			if (devicesSuccess > 0 && devicesFailed === 0) {
				showToast('Foto actualizada y sincronizada');
			} else if (devicesSuccess > 0) {
				showToast('Foto guardada; algunos dispositivos no confirmaron el rostro', 'error');
			} else if (devicesTotal > 0) {
				showToast(result.error || 'Foto guardada localmente, pero el dispositivo no acepto el rostro', 'error');
			} else {
				showToast('Foto actualizada');
			}
			document.getElementById('employee-modal')?.classList.remove('active');
			await loadEmployees();
		}
		return true;
	} catch (err) {
		showToast('Error de conexion al subir la foto', 'error');
		return false;
	}
};

async function initPWA() {
    if (!('serviceWorker' in navigator)) {
        return;
    }

    try {
        await navigator.serviceWorker.register('/service-worker.js');
        console.log('PWA service worker registered');
    } catch (err) {
        console.warn('PWA service worker registration failed', err);
    }
}

function initConfig() {
    const form = document.getElementById('config-form');
    if (!form) return;

    // Load config when viewing settings page
    loadConfig();

    const btnConsult = document.getElementById('btn-consult-rnc');
    if (btnConsult) {
        btnConsult.addEventListener('click', async () => {
            const rncInput = document.getElementById('config-company-rnc');
            const rnc = rncInput.value.trim().replace(/-/g, '');
            if (!rnc) {
                showToast('Ingresa un RNC para consultar', 'warning');
                return;
            }

            btnConsult.disabled = true;
            btnConsult.innerHTML = '<span class="loading-spinner--xs"></span> Buscando...';

            try {
                // Consult using the backend proxy to avoid CORS/Connection issues
                const resp = await fetch('/api/config/rnc-lookup', {
                    method: 'POST',
                    headers: getAuthHeaders(),
                    body: JSON.stringify({ RNC: rnc })
                });

                if (resp.ok) {
                    const result = await resp.json();
                    
                    // The API response might be an array or an object
                    const data = Array.isArray(result) ? result[0] : result;
                    
                    // Mapping common field names for name/reason social from Dominican DGII APIs
                    const companyName = data.Nombre || data.RazonSocial || data.NOMBRE_COMERCIAL || data.RAZON_SOCIAL || data.nombre || '';
                    
                    if (companyName) {
                        const nameInput = form.querySelector('[name="company_name"]');
                        if (nameInput) {
                            nameInput.value = companyName;
                            showToast('Empresa encontrada: ' + companyName);
                        }
                    } else {
                        showToast('No se encontró información para este RNC', 'warning');
                    }
                } else {
                    showToast('Error en la consulta. Verifica el RNC.', 'error');
                }
            } catch (err) {
                console.error('RNC consultation failed', err);
                showToast('Fallo en la conexión con la API de consulta', 'error');
            } finally {
                btnConsult.disabled = false;
                btnConsult.innerHTML = '<svg style="width:16px;height:16px;"><use href="#icon-refresh"></use></svg> Consultar RNC';
            }
        });
    }

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const rawData = Object.fromEntries(formData.entries());
        // Extract weekly schedule
        const weeklySchedule = {};
        const tbody = document.querySelector('#weekly-schedule-table tbody');
        if (tbody) {
            tbody.querySelectorAll('tr').forEach(tr => {
                const dayKey = tr.dataset.day;
                const isWorkday = tr.querySelector('.is-workday').checked;
                const start = tr.querySelector('.start-time').value;
                const end = tr.querySelector('.end-time').value;
                weeklySchedule[dayKey] = { is_workday: isWorkday, start, end };
            });
        }

        const data = {
            company_name: rawData.company_name || '',
            company_rnc: rawData.company_rnc || '',
            grace_period_minutes: rawData.grace_period || '',
            overtime_threshold_hours: rawData.work_hours || '',
            default_shift_start: rawData.default_shift_start || '08:00',
            default_shift_end: rawData.default_shift_end || '17:00',
            weekly_schedule: JSON.stringify(weeklySchedule),
            travel_module_enabled: form.querySelector('[name="travel_module_enabled"]').checked ? 'true' : 'false'
        };

        try {
            const resp = await fetch('/api/config', {
                method: 'POST',
                headers: getAuthHeaders(),
                body: JSON.stringify(data)
            });

            const result = await resp.json();

            if (resp.ok) {
                showToast(result.message || 'Configuración guardada y aplicada');
                // Update local state and UI
                window.isTravelModuleEnabled = data.travel_module_enabled === 'true';
                applyRolePermissions();
            } else {
                showToast(result.error || 'Error al guardar configuración', 'error');
            }
        } catch (err) {
            showToast('Error de conexión', 'error');
        }
    });
}

// Load configuration from server
async function loadConfig() {
    try {
        const resp = await fetch('/api/config', {
            headers: getAuthHeaders()
        });
        if (!resp.ok) return;

        const config = await resp.json();

        // Populate form fields
        const form = document.getElementById('config-form');
        if (!form) return;

        // Map API keys to form field names
        const fieldMap = {
            'company_name': 'company_name',
            'company_rnc': 'company_rnc',
            'grace_period_minutes': 'grace_period',
            'overtime_threshold_hours': 'work_hours',
            'default_shift_start': 'default_shift_start',
            'default_shift_end': 'default_shift_end',
            'travel_module_enabled': 'travel_module_enabled',
        };

        for (const [apiKey, fieldName] of Object.entries(fieldMap)) {
            const field = form.querySelector(`[name="${fieldName}"]`);
            if (field && config[apiKey] !== undefined) {
                if (field.type === 'checkbox') {
                    field.checked = String(config[apiKey]) === 'true';
                } else {
                    field.value = config[apiKey];
                }
            }
        }

        // Set global state for feature toggles
        window.isTravelModuleEnabled = config.travel_module_enabled === 'true';
        applyRolePermissions();

        // Render Weekly Schedule
        const days = [
            { key: 'Monday', name: 'Lunes' },
            { key: 'Tuesday', name: 'Martes' },
            { key: 'Wednesday', name: 'Miércoles' },
            { key: 'Thursday', name: 'Jueves' },
            { key: 'Friday', name: 'Viernes' },
            { key: 'Saturday', name: 'Sábado' },
            { key: 'Sunday', name: 'Domingo' }
        ];
        
        let weeklySch = {};
        try {
            if (config.weekly_schedule) weeklySch = JSON.parse(config.weekly_schedule);
        } catch(e) {}

        const tbody = document.querySelector('#weekly-schedule-table tbody');
        if (tbody) {
            tbody.innerHTML = '';
            days.forEach(d => {
                const dayConfig = weeklySch[d.key] || { 
                    is_workday: d.key !== 'Sunday' && d.key !== 'Saturday', 
                    start: config.default_shift_start || '08:00', 
                    end: config.default_shift_end || '17:00' 
                };
                
                const tr = document.createElement('tr');
                tr.dataset.day = d.key;
                tr.innerHTML = `
                    <td style="font-weight: 500; font-size: 0.9em; padding: 8px;">${d.name}</td>
                    <td style="text-align: center; padding: 8px;">
                        <label class="checkbox-container" style="display: inline-block; padding-left: 20px; margin-bottom: 0;">
                            <input type="checkbox" class="is-workday" ${dayConfig.is_workday ? 'checked' : ''}>
                            <span class="checkmark"></span>
                        </label>
                    </td>
                    <td style="text-align: center; padding: 8px;">
                        <input type="time" class="start-time" value="${dayConfig.start}" ${!dayConfig.is_workday ? 'disabled' : ''} style="width: 100%; min-width: 100px; max-width: 130px; text-align: center; padding: 6px;">
                    </td>
                    <td style="text-align: center; padding: 8px;">
                        <input type="time" class="end-time" value="${dayConfig.end}" ${!dayConfig.is_workday ? 'disabled' : ''} style="width: 100%; min-width: 100px; max-width: 130px; text-align: center; padding: 6px;">
                    </td>
                `;
                
                // Toggle inputs when checkbox changes
                const checkbox = tr.querySelector('.is-workday');
                const timeInputs = tr.querySelectorAll('input[type="time"]');
                checkbox.addEventListener('change', (e) => {
                    timeInputs.forEach(inp => inp.disabled = !e.target.checked);
                });

                tbody.appendChild(tr);
            });
        }
    } catch (err) {
        console.error('Failed to load config', err);
    }
}

function initLDAP() {
    const form = document.getElementById('ldap-form');
    const btnTest = document.getElementById('btn-test-ldap');
    const btnSync = document.getElementById('btn-sync-ldap');
    if (!form) return;

    // Load LDAP config
    loadLDAPConfig();

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());

        try {
            const resp = await fetch('/api/config', {
                method: 'POST',
                headers: getAuthHeaders(),
                body: JSON.stringify(data)
            });

            if (resp.ok) {
                showToast('Configuración LDAP guardada');
            } else {
                showToast('Error al guardar configuración', 'error');
            }
        } catch (err) {
            showToast('Error de conexión', 'error');
        }
    });

    btnTest.addEventListener('click', async () => {
        const originalText = btnTest.innerText;
        btnTest.disabled = true;
        btnTest.innerText = 'Probando...';

        try {
            const resp = await fetch('/api/ldap/test', {
                method: 'POST',
                headers: getAuthHeaders()
            });
            if (resp.ok) {
                const data = await resp.json();
                showToast(data.message || 'Conexión exitosa');
            } else {
                showToast(result.error || 'No se pudo registrar el rostro', 'error');
            }
        } catch (err) {
            showToast('Error de conexión', 'error');
        } finally {
            btnTest.disabled = false;
            btnTest.innerText = originalText;
        }
    });

    btnSync.addEventListener('click', async () => {
        const originalText = btnSync.innerText;
        btnSync.disabled = true;
        btnSync.innerText = 'Sincronizando...';

        try {
            const resp = await fetch('/api/ldap/sync', {
                method: 'POST',
                headers: getAuthHeaders()
            });
            if (resp.ok) {
                const data = await resp.json();
                showToast(data.message || 'Sincronización completada');
                // Si la página actual es empleados, recargar
                const activeNav = document.querySelector('.sidebar nav li.active');
                if (activeNav && activeNav.getAttribute('data-page') === 'employees') {
                    loadEmployees();
                }
            } else {
                showToast(result.error || 'No se pudo registrar el rostro', 'error');
            }
        } catch (err) {
            showToast('Error de conexión', 'error');
        } finally {
            btnSync.disabled = false;
            btnSync.innerText = originalText;
        }
    });
}

// Load LDAP configuration
async function loadLDAPConfig() {
    try {
        const resp = await fetch('/api/config', {
            headers: getAuthHeaders()
        });
        if (!resp.ok) return;

        const config = await resp.json();

        // Populate LDAP form fields
        const form = document.getElementById('ldap-form');
        if (!form) return;

        const ldapFields = {
            'ldap_host': 'ldap_host',
            'ldap_port': 'ldap_port',
            'ldap_base_dn': 'ldap_base_dn',
            'ldap_bind_dn': 'ldap_bind_dn',
            'ldap_user_filter': 'ldap_user_filter',
        };

        for (const [apiKey, fieldName] of Object.entries(ldapFields)) {
            const field = form.querySelector(`[name="${fieldName}"]`);
            if (field && config[apiKey] !== undefined && config[apiKey] !== '') {
                field.value = config[apiKey];
            }
        }
    } catch (err) {
        console.error('Failed to load LDAP config', err);
    }
}

function initFaceUI() {
    const modal = document.getElementById('face-modal');
    const form = document.getElementById('face-form');
    const closeBtns = modal.querySelectorAll('.close-modal');
    const btnImportFace = document.getElementById('btn-import-face');

    closeBtns.forEach(btn => btn.addEventListener('click', () => modal.classList.remove('active')));

    if (btnImportFace) {
        btnImportFace.addEventListener('click', async () => {
            const empNo = document.getElementById('face-emp-no').value;
            if (!empNo) return;
            
            btnImportFace.disabled = true;
            btnImportFace.innerText = 'Importando...';
            showToast('Recuperando imagen del dispositivo...');
            
            try {
                const resp = await fetch(`/api/employees/${empNo}/face/import`, {
                    method: 'POST',
                    headers: getAuthHeaders()
                });
                const result = await resp.json();
                
                if (resp.ok) {
                    showToast('Imagen recuperada y guardada localmente');
                    modal.classList.remove('active');
                    // Refresh employee list or detail if visible
                    if (window.currentEditingEmployee && window.currentEditingEmployee.employeeNo === empNo) {
                        editEmployee(window.currentEditingEmployee.id);
                    } else {
                        loadEmployees();
                    }
                } else {
                    showToast(result.error || 'No se encontró rostro en el dispositivo', 'error');
                }
            } catch (err) {
                showToast('Error de conexión', 'error');
            } finally {
                btnImportFace.disabled = false;
                btnImportFace.innerText = 'Importar del dispositivo';
            }
        });
    }

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const empNo = document.getElementById('face-emp-no').value;
        const formData = new FormData(form);

        const btn = form.querySelector('button[type="submit"]');
        btn.disabled = true;
        btn.innerText = 'Enviando...';

        try {
            const token = currentUser?.token || sessionStorage.getItem('token') || localStorage.getItem('auth_token');
            const resp = await fetch(`/api/employees/${empNo}/face`, {
                method: 'POST',
                headers: token ? { 'Authorization': `Bearer ${token}` } : {},
                body: formData // Multipart
            });
            const result = await safeReadJSON(resp);

            if (resp.ok) {
                showToast('Rostro registrado con éxito en el terminal');
                modal.classList.remove('active');
            } else {
                const err = await resp.text();
                showToast(`Error: ${err}`, 'error');
            }
        } catch (err) {
            showToast('Error de conexión', 'error');
        } finally {
            btn.disabled = false;
            btn.innerText = 'Enviar al Dispositivo';
        }
    });
}

window.openFaceModal = (empNo) => {
    document.getElementById('face-emp-no').value = empNo;
    const btnImportFace = document.getElementById('btn-import-face');
    
    // Si el empleado ya tiene foto local, quizás no necesitemos importar, 
    // pero permitimos al usuario forzar la recuperación si lo desea.
    if (btnImportFace) {
        btnImportFace.style.display = 'block';
    }
    
    document.getElementById('face-modal').classList.add('active');
};

window.editEmployee = async (id) => {
    try {
        const resp = await fetch(`/api/employees/${id}`, {
            headers: getAuthHeaders()
        });
        if (!resp.ok) {
            const result = await safeReadJSON(resp);
            showToast(result.error || 'No se pudo cargar el empleado', 'error');
            return;
        }

        const employee = await resp.json();
        const form = document.getElementById('employee-form');
        await loadEmployeeFormOptions();
        resetEmployeeForm(form);
        window.currentEditingEmployee = employee;
        updateEmployeeAdminUI(employee);

        document.getElementById('employee-id').value = employee.id || '';
        form.querySelector('[name="firstName"]').value = employee.firstName || '';
        form.querySelector('[name="lastName"]').value = employee.lastName || '';
        form.querySelector('[name="idNumber"]').value = employee.idNumber || '';
        form.querySelector('[name="employeeNo"]').value = employee.employeeNo || '';
        form.querySelector('[name="cardNo"]').value = employee.cardNo || '';
        form.querySelector('[name="fleetNo"]').value = employee.fleetNo || '';
        form.querySelector('[name="personalNo"]').value = employee.personalNo || '';
        form.querySelector('[name="email"]').value = employee.email || '';
        form.querySelector('[name="baseSalary"]').value = employee.baseSalary || '';
        form.querySelector('[name="birthDate"]').value = toDateInputValue(employee.birthDate);
        form.querySelector('[name="hireDate"]').value = toDateInputValue(employee.hireDate);
        form.querySelector('[name="departmentId"]').value = employee.departmentId || '';
        form.querySelector('[name="positionId"]').value = employee.positionId || '';
        form.querySelector('[name="status"]').value = employee.status || 'Active';

        document.getElementById('modal-title').innerText = 'Editar Empleado';
        const submitBtn = document.getElementById('employee-submit-btn');
        if (submitBtn) {
            submitBtn.innerText = 'Actualizar Empleado';
        }

        // Show photo if available
        const photoUrlInput = document.getElementById('employee-photo-url');
        const btnImportFace = document.getElementById('btn-import-face');
        
        if (photoUrlInput) photoUrlInput.value = employee.photoUrl || '';
        window.employeePhotoRemoved = false;

        const src = employeePhotoSrc(employee);
        if (src) {
            setEmployeePhotoState(src);
        } else {
            setEmployeePhotoState('', 'Sin foto local');
        }
        updateEmployeeAdminUI(employee);

        document.getElementById('employee-modal').classList.add('active');
        return;
    } catch (err) {
        showToast('Error de conexion', 'error');
        return;
    }
    showToast('Función de editar pendiente', 'info');
};

function toDateInputValue(value) {
    if (!value) return '';
    return value.split('T')[0];
}

async function downloadReport(url, filename) {
    try {
        const resp = await fetch(url);

        if (!resp.ok) {
            const errText = await resp.text();
            showToast(`Error: ${errText || 'Falló la descarga del reporte'}`, 'error');
            return;
        }

        const blob = await resp.blob();
        const downloadUrl = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = downloadUrl;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(downloadUrl);
        showToast('Reporte descargado exitosamente');
    } catch (err) {
        showToast('Error al descargar reporte: ' + err.message, 'error');
    }
}

function initReports() {
    const reportGrid = document.querySelector('.reports-grid');
    if (!reportGrid) return;

    reportGrid.addEventListener('click', (e) => {
        if (e.target.classList.contains('btn-outline')) {
            const format = e.target.innerText.toLowerCase(); // pdf or excel
            const reportType = e.target.closest('.report-card').querySelector('h4').innerText;

            // Get date range (default to current month)
            const now = new Date();
            const from = now.toISOString().split('T')[0];
            const to = from;

            const ext = format === 'excel' ? 'xlsx' : 'pdf';

            if (reportType === 'Asistencia Diaria') {
                downloadReport(`/api/reports/daily?format=${format}&date=${from}`, `reporte_diario_${from}.${ext}`);
            } else if (reportType === 'Pre-Nómina Quincenal') {
                downloadReport(`/api/reports/payroll?format=${format}&from=${from}&to=${to}`, `prenomina_${from}_${to}.xlsx`);
            } else if (reportType === 'Tardanzas y Faltas') {
                downloadReport(`/api/reports/late?format=${format}&from=${from}&to=${to}`, `reporte_tardanzas_${from}_${to}.${ext}`);
            } else if (reportType === 'KPIs de Asistencia') {
                // KPIs only available in Excel
                downloadReport(`/api/reports/kpis?format=excel&from=${from}&to=${to}`, `reporte_kpis_${from}_${to}.xlsx`);
            } else {
                showToast('Tipo de reporte no implementado aún', 'error');
            }
        }
    });
}

function initEmployeeUI() {
    const modal = document.getElementById('employee-modal');
    const btnAdd = document.getElementById('btn-add-employee');
    const closeBtns = document.querySelectorAll('.close-modal');
    const form = document.getElementById('employee-form');
    const submitBtn = document.getElementById('employee-submit-btn');

    if (btnAdd) {
        btnAdd.addEventListener('click', async () => {
            resetEmployeeForm(form);
            window.currentEditingEmployee = null;
            updateEmployeeAdminUI(null);
            document.getElementById('modal-title').innerText = 'Nuevo Empleado';
            if (submitBtn) {
                submitBtn.innerText = 'Guardar Empleado';
            }
            setEmployeePhotoState('');
            
            await loadEmployeeFormOptions();
            modal.classList.add('active');
        });
    }

    const btnSyncEmployees = document.getElementById('btn-sync-employees');
    if (btnSyncEmployees) {
        btnSyncEmployees.addEventListener('click', () => syncDeviceEmployees('default'));
    }

    closeBtns.forEach(btn => {
        btn.addEventListener('click', () => modal.classList.remove('active'));
    });

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());
        const employeeId = data.id;
        delete data.id;

        data.baseSalary = data.baseSalary ? parseFloat(data.baseSalary) : 0;
        data.status = data.status || 'Active';
        data.photoRemoved = Boolean(employeeId && window.employeePhotoRemoved);

        try {
            const url = employeeId ? `/api/employees/${employeeId}` : '/api/employees';
            const method = employeeId ? 'PUT' : 'POST';
            
            const resp = await fetch(url, {
                method: method,
                headers: getAuthHeaders(),
                body: JSON.stringify(data)
            });

            if (resp.ok) {
                const savedEmp = await resp.json();
                const empNo = data.employeeNo || savedEmp.employeeNo;
                if (window.currentEmployeePhoto && empNo) {
                    const uploaded = await uploadEmployeePhoto(empNo, { silent: true });
                    if (!uploaded) return;
                }
                showToast(employeeId ? 'Empleado actualizado correctamente' : 'Empleado guardado correctamente');
                modal.classList.remove('active');
                loadEmployees();
            } else {
                const result = await safeReadJSON(resp);
                showToast(result.error || 'Error al guardar empleado', 'error');
            }
        } catch (err) {
            showToast('Error de conexión', 'error');
        }
    });
}

function resetEmployeeForm(form) {
    form.reset();
    window.currentEmployeePhoto = null;
    window.currentEditingEmployee = null;
    window.employeePhotoRemoved = false;
    updateEmployeeAdminUI(null);
    document.getElementById('employee-id').value = '';
    const photoUrlInput = document.getElementById('employee-photo-url');
    if (photoUrlInput) photoUrlInput.value = '';
    
    const btnImportFace = document.getElementById('btn-import-face');
    setEmployeePhotoState('');
    const photoInput = document.getElementById('input-employee-photo');
    if (photoInput) photoInput.value = '';
    if (btnImportFace) btnImportFace.style.display = 'none';

    const statusField = form.querySelector('[name="status"]');
    if (statusField) {
        statusField.value = 'Active';
    }
}

window.deleteEmployeePhoto = async function(employeeNo, options = {}) {
	if (!employeeNo) {
		showToast('No se encontro el numero de empleado para borrar la foto', 'error');
		return false;
	}

	try {
		const resp = await fetch(`/api/employees/${encodeURIComponent(employeeNo)}/photo`, {
			method: 'DELETE',
			headers: getAuthHeaders()
		});

		if (!resp.ok) {
			const result = await safeReadJSON(resp);
			showToast(result.error || 'No se pudo borrar la foto', 'error');
			return false;
		}

		window.currentEmployeePhoto = null;
		window.employeePhotoRemoved = false;
		if (window.currentEditingEmployee) {
			window.currentEditingEmployee.photoUrl = '';
			window.currentEditingEmployee.photoData = null;
		}

		const input = document.getElementById('input-employee-photo');
		const photoUrlInput = document.getElementById('employee-photo-url');
		if (input) input.value = '';
		if (photoUrlInput) photoUrlInput.value = '';

		if (!options.silent) {
			showToast('Foto eliminada');
			await loadEmployees();
		}
		return true;
	} catch (err) {
		showToast('Error de conexion al borrar la foto', 'error');
		return false;
	}
};

async function loadEmployeeFormOptions() {
    await Promise.all([
        loadEmployeeDepartments(),
        loadEmployeePositions()
    ]);
}

async function safeReadJSON(resp) {
    try {
        return await resp.json();
    } catch (err) {
        return {};
    }
}

function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    toast.innerText = message;
    toast.classList.remove('toast-success', 'toast-error');
    toast.classList.add(type === 'success' ? 'toast-success' : 'toast-error');
    toast.classList.add('active');
    setTimeout(() => toast.classList.remove('active'), 3000);
}

function initNavigation() {
    const navItems = document.querySelectorAll('.sidebar nav li:not(.nav-label):not(.nav-item-org)');
    const orgItems = document.querySelectorAll('.nav-item-org');
    const pages = document.querySelectorAll('.page');
    const pageTitle = document.getElementById('page-title');

    // Handle Organization items (Empleados, Departamentos, Cargos, Asistencia)
    orgItems.forEach(item => {
        item.addEventListener('click', () => {
            const pageId = item.getAttribute('data-page');

            // Update active states
            document.querySelectorAll('.sidebar nav li').forEach(i => i.classList.remove('active'));
            item.classList.add('active');

            pages.forEach(p => p.classList.remove('active'));
            const activePage = document.getElementById(pageId);
            if (activePage) activePage.classList.add('active');

            pageTitle.innerText = item.dataset.title || item.textContent.trim();
            closeMobileMenu();

            // Load data based on page
            if (pageId === 'employees') loadEmployees();
            if (pageId === 'departments') loadDepartments();
            if (pageId === 'positions') loadPositions();
            if (pageId === 'travel-allowances') loadTravelAllowances();
            if (pageId === 'leaves') loadLeaves();
            if (pageId === 'holidays') loadHolidays();
            if (pageId === 'audit') AuditLogManager.render();

            document.querySelector('.content').scrollTop = 0;
        });
    });

    const refreshAuditBtn = document.getElementById('btn-refresh-audit');
    if (refreshAuditBtn) {
        refreshAuditBtn.onclick = () => AuditLogManager.render();
    }


    // Handle regular nav items
    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const pageId = item.getAttribute('data-page');

            navItems.forEach(i => i.classList.remove('active'));
            item.classList.add('active');
            orgItems.forEach(i => i.classList.remove('active'));

            pages.forEach(p => p.classList.remove('active'));
            const activePage = document.getElementById(pageId);
            if (activePage) activePage.classList.add('active');

            pageTitle.innerText = item.dataset.title || item.textContent.trim();
            closeMobileMenu();

            // Cargar datos según la página
            if (pageId === 'dashboard') loadDashboardStats();
            if (pageId === 'devices') loadManagedDevices();
            if (pageId === 'settings') DeviceErrorManager.fetchLogs().then(logs => DeviceErrorManager.renderLogs(logs));
            if (pageId === 'attendance' && window.loadAttendance) window.loadAttendance();

            document.querySelector('.content').scrollTop = 0;
        });
    });
}

function initMobileMenu() {
    const button = document.getElementById('btn-mobile-menu');
    const sidebar = document.querySelector('.sidebar');
    const backdrop = document.getElementById('sidebar-backdrop');

    if (!button || !sidebar || !backdrop) return;

    button.addEventListener('click', () => {
        const isOpen = sidebar.classList.toggle('mobile-open');
        backdrop.classList.toggle('active', isOpen);
        document.body.classList.toggle('menu-open', isOpen);
    });

    backdrop.addEventListener('click', closeMobileMenu);
    window.addEventListener('resize', () => {
        if (window.innerWidth > 960) {
            closeMobileMenu();
        }
    });
}

function closeMobileMenu() {
    const sidebar = document.querySelector('.sidebar');
    const backdrop = document.getElementById('sidebar-backdrop');
    if (!sidebar || !backdrop) return;

    sidebar.classList.remove('mobile-open');
    backdrop.classList.remove('active');
    document.body.classList.remove('menu-open');
    const supportWrap = document.getElementById('support-fab-wrap');
    if (supportWrap) {
        supportWrap.classList.remove('open');
    }
}

async function loadDevices() {
    await loadManagedDevices();
    renderDevices(discoveredDevices);
}

function initWebSocket() {
    if (!currentUser) {
        return;
    }
    if (wsSocket && (wsSocket.readyState === WebSocket.OPEN || wsSocket.readyState === WebSocket.CONNECTING)) {
        return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = currentUser?.token || sessionStorage.getItem('token') || localStorage.getItem('auth_token');
    if (!token) {
        return;
    }
    const wsUrl = `${protocol}//${window.location.host}/ws?token=${encodeURIComponent(token)}`;
    wsSocket = new WebSocket(wsUrl);

    wsSocket.onopen = () => {
        console.log('Real-time connection established');
        if (wsReconnectTimer) {
            clearTimeout(wsReconnectTimer);
            wsReconnectTimer = null;
        }
    };

    wsSocket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.type === 'attendance') {
            addEventToTable(data);
            // Also update dashboard table if we are on dashboard
            addEventToDashboardTable(data);
            // Refresh stats to update counters
            scheduleDashboardStatsRefresh();
        }
    };

    wsSocket.onclose = () => {
        wsSocket = null;
        if (!currentUser || wsReconnectTimer) {
            return;
        }
        console.warn('Real-time connection lost. Retrying in 5s...');
        wsReconnectTimer = setTimeout(() => {
            wsReconnectTimer = null;
            initWebSocket();
        }, 5000);
    };
}

function addEventToTable(event) {
    const tbody = document.querySelector('#events-table tbody');
    if (!tbody) return;

    const payload = event.data || event;
    const row = document.createElement('tr');
    
    // Format timestamp
    const date = new Date(payload.dateTime || payload.timestamp || Date.now());
    const timeStr = date.toLocaleTimeString();

    row.innerHTML = `
        <td>${timeStr}</td>
        <td>${escapeHTML(payload.employeeName || payload.name || '---')}</td>
        <td>${escapeHTML(payload.employeeNo || '---')}</td>
        <td>${escapeHTML(payload.deviceId || payload.ipAddress || 'Device')}</td>
        <td><span class="badge badge-success">Acceso</span></td>
    `;

    tbody.prepend(row);
    if (tbody.children.length > 20) tbody.removeChild(tbody.lastChild);
}

function addEventToDashboardTable(event) {
    const tbody = document.querySelector('#dashboard-events-table tbody');
    if (!tbody) return;

    // Remove empty state if present
    if (tbody.querySelector('td[colspan]')) {
        tbody.innerHTML = '';
    }

    const payload = event.data || event;
    const row = document.createElement('tr');
    
    const date = new Date(payload.dateTime || payload.timestamp || Date.now());
    const timeStr = date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});

    row.innerHTML = `
        <td>${timeStr}</td>
        <td>
            <div style="display:flex; flex-direction:column;">
                <strong>${escapeHTML(payload.employeeName || '---')}</strong>
                <span style="font-size:0.7rem; color:var(--text-muted);">ID: ${payload.employeeNo || '---'}</span>
            </div>
        </td>
        <td><span class="badge badge-secondary" style="font-size:0.65rem;">${escapeHTML(payload.deviceId || 'Device')}</span></td>
    `;

    tbody.prepend(row);
    if (tbody.children.length > 10) tbody.removeChild(tbody.lastChild);
}

async function initScan() {
    const scanButtons = ['btn-scan', 'btn-refresh-devices', 'btn-scan-devices-inline'];

    scanButtons.forEach(id => {
        const btn = document.getElementById(id);
        if (!btn) return;

        btn.addEventListener('click', async () => {
            const originalText = btn.innerText;
            btn.disabled = true;
            btn.innerText = 'Escaneando...';

            try {
                const resp = await fetch('/api/discovery/discover', {
                    method: 'GET',
                    headers: getAuthHeaders()
                });
                const devices = await resp.json();
                discoveredDevices = Array.isArray(devices) ? devices : [];

                if (discoveredDevices.length > 0) {
                    renderDevices(discoveredDevices);
                    showToast(`Se encontraron ${discoveredDevices.length} dispositivos Hikvision.`);
                } else {
                    renderDevices([]);
                    showToast('No se encontraron dispositivos en la red local.', 'error');
                }
            } catch (err) {
                console.error('Scan failed', err);
                showToast('Error al escanear la red.', 'error');
            } finally {
                btn.disabled = false;
                btn.innerText = originalText;
            }
        });
    });
}

function renderDevices(devices) {
    const tbody = document.getElementById('discovered-devices-list');
    if (!tbody) return;

    if (!devices || devices.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-muted" style="text-align: center; padding: 2rem;">Sin resultados recientes de SADP. Usa "Escanear ahora" para detectar terminales.</td></tr>';
        return;
    }

    tbody.innerHTML = devices.map(dev => `
        <tr>
            <td>${escapeHTML(dev.DeviceType || 'Hikvision')}</td>
            <td>${escapeHTML(dev.DeviceSN || '---')}</td>
            <td><a href="http://${dev.IPv4Address}" target="_blank" class="device-ip-link">${escapeHTML(dev.IPv4Address || '---')}</a></td>
            <td><span class="badge badge-secondary">SADP</span></td>
            <td>
                <button class="btn btn-sm" onclick="createManagedDeviceFromScan(decodeInlineArg('${encodeInlineArg(dev.IPv4Address || '')}'), decodeInlineArg('${encodeInlineArg(dev.DeviceType || 'Hikvision')}'), decodeInlineArg('${encodeInlineArg(dev.DeviceSN || '')}'))">Agregar</button>
            </td>
        </tr>
    `).join('');
}

function initDeviceManager() {
    const modal = document.getElementById('device-modal');
    const form = document.getElementById('device-form');
    const addButtons = ['btn-add-device', 'btn-add-device-inline'];

    if (!modal || !form) return;

    addButtons.forEach(id => {
        const btn = document.getElementById(id);
        if (!btn) return;
        btn.addEventListener('click', () => openDeviceModal());
    });

    const btnRefreshStatus = document.getElementById('btn-refresh-devices-status');
    if (btnRefreshStatus) {
        btnRefreshStatus.addEventListener('click', () => loadManagedDevices());
    }

    const btnSyncAll = document.getElementById('btn-sync-all-config');
    if (btnSyncAll) {
        btnSyncAll.addEventListener('click', () => syncDeviceEmployees('default'));
    }

    const btnReadEvents = document.getElementById('btn-read-recent-events');
    if (btnReadEvents) {
        btnReadEvents.addEventListener('click', async () => {
            btnReadEvents.disabled = true;
            btnReadEvents.innerText = 'Leyendo...';
            showToast('Leyendo eventos de dispositivos...');
            try {
                const resp = await fetch('/api/devices/read-events', {
                    method: 'POST',
                    headers: getAuthHeaders()
                });
                const result = await resp.json();
                if (resp.ok) {
                    showToast(`Se leyeron ${result.eventsRead} eventos nuevos.`);
                } else {
                    showToast(result.error || 'Error al leer eventos', 'error');
                }
            } catch (err) {
                showToast('Error de conexión', 'error');
            } finally {
                btnReadEvents.disabled = false;
                btnReadEvents.innerText = 'Leer Ponches Recientes';
            }
        });
    }

    const btnImportAllPhotos = document.getElementById('btn-import-all-photos');
    if (btnImportAllPhotos) {
        btnImportAllPhotos.addEventListener('click', async () => {
            if (!confirm('¿Desea intentar importar las fotos de TODOS los empleados que no tienen imagen local? Este proceso puede tardar unos minutos.')) return;
            
            btnImportAllPhotos.disabled = true;
            const originalText = btnImportAllPhotos.innerText;
            btnImportAllPhotos.innerText = 'Importando...';
            showToast('Iniciando importación masiva de fotos...');
            
            try {
                const resp = await fetch('/api/devices/import-photos', {
                    method: 'POST',
                    headers: getAuthHeaders()
                });
                const result = await resp.json();
                
                if (resp.ok) {
                    showToast(`Proceso completado. Éxitos: ${result.imported}, Fallos: ${result.failed}`);
                    loadEmployees();
                } else {
                    showToast(result.error || 'Error en la importación masiva', 'error');
                }
            } catch (err) {
                showToast('Error de conexión', 'error');
            } finally {
                btnImportAllPhotos.disabled = false;
                btnImportAllPhotos.innerText = originalText;
            }
        });
    }

    modal.querySelectorAll('.close-modal').forEach(btn => {
        btn.addEventListener('click', () => modal.classList.remove('active'));
    });

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());
        const deviceId = data.id;
        delete data.id;
        data.port = parseInt(data.port, 10) || 80;
        data.isDefault = document.getElementById('device-default').checked;

        if (!data.password && deviceId) {
            delete data.password;
        }

        try {
            const resp = await fetch(deviceId ? `/api/devices/configured/${deviceId}` : '/api/devices/configured', {
                method: deviceId ? 'PUT' : 'POST',
                headers: getAuthHeaders(),
                body: JSON.stringify(data)
            });

            if (!resp.ok) {
                const err = await resp.text();
                showToast(`Error: ${err}`, 'error');
                return;
            }

            showToast(deviceId ? 'Dispositivo actualizado' : 'Dispositivo agregado');
            modal.classList.remove('active');
            await loadManagedDevices();
        } catch (err) {
            showToast('Error de conexión', 'error');
        }
    });
}

async function loadManagedDevices() {
    try {
        const resp = await fetch('/api/devices/configured', {
            headers: getAuthHeaders()
        });
        if (!resp.ok) {
            return;
        }

        managedDevices = await resp.json();
        renderManagedDevices();
    } catch (err) {
        console.error('Load managed devices failed', err);
    }
}

function renderManagedDevices() {
    const tbody = document.getElementById('managed-devices-list');
    if (!tbody) return;

    const countEl = document.getElementById('managed-devices-count');
    const defaultEl = document.getElementById('default-device-name');
    const dashboardDeviceState = document.getElementById('dashboard-device-state');
    const defaultDevice = managedDevices.find(device => device.isDefault);

    if (countEl) countEl.innerText = managedDevices.length;
    if (defaultEl) defaultEl.innerText = defaultDevice ? defaultDevice.name : 'Sin asignar';
    if (document.getElementById('stats-devices')) {
        document.getElementById('stats-devices').innerText = managedDevices.length;
    }
    if (dashboardDeviceState) {
        dashboardDeviceState.innerText = defaultDevice
            ? `Predeterminado: ${defaultDevice.name}`
            : 'Sin terminal predeterminada';
    }

    if (!managedDevices || managedDevices.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-muted" style="text-align: center; padding: 2rem;">Aún no hay dispositivos administrados. Agrega uno manualmente.</td></tr>';
        return;
    }

    tbody.innerHTML = managedDevices.map(device => `
        <tr class="${device.isOnline ? '' : 'device-row--offline'}">
            <td>
                <div class="device-identity">
                    <strong>${escapeHTML(device.name || '')}</strong>
                    <span class="text-muted">${escapeHTML(device.model || 'Terminal Hikvision')} ${!device.isOnline ? '<span class="text-danger" style="font-size:0.7rem; margin-left:5px;">(Fuera de línea)</span>' : ''}</span>
                </div>
            </td>
            <td><a href="http://${device.ip}:${device.port || 80}" target="_blank" class="device-ip-link">${escapeHTML(device.ip || '')}:${device.port || 80}</a></td>
            <td>
                <div style="display: flex; flex-direction: column; gap: 4px;">
                    <span class="badge ${device.isOnline ? 'badge-success' : 'badge-danger'}" title="${device.error || ''}">
                        ${device.isOnline ? '● Conectado' : '● Desconectado'}
                    </span>
                    ${!device.isOnline && device.error ? `<span class="text-muted" style="font-size: 0.65rem; max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${escapeHTML(device.error)}">${escapeHTML(device.error)}</span>` : ''}
                </div>
            </td>
            <td>${escapeHTML(device.username || '---')} ${device.hasPassword ? '<span class="badge badge-success">OK</span>' : '<span class="badge badge-warning">Sin clave</span>'}</td>
            <td>${device.isDefault ? '<span class="badge badge-success">Predeterminado</span>' : '<span class="badge badge-secondary">Secundario</span>'}</td>
            <td>
                    <div class="travel-actions" style="justify-content: flex-end;">
                        <button class="btn-action btn-action--icon btn-action--view" onclick="syncDeviceTime(decodeInlineArg('${encodeInlineArg(device.id || '')}'))" title="Sincronizar Hora del Dispositivo"><svg style="width:16px;height:16px;"><use href="#icon-calendar"></use></svg></button>
                        <button class="btn-action btn-action--icon btn-action--view" onclick="setupAlarmHost(decodeInlineArg('${encodeInlineArg(device.id || '')}'))" title="Configurar Escucha en Tiempo Real (Alarm Host)"><svg style="width:16px;height:16px;"><use href="#icon-wifi"></use></svg></button>
                        <button class="btn-action btn-action--icon btn-action--view" onclick="syncDeviceEmployees(decodeInlineArg('${encodeInlineArg(device.id || '')}'))" title="Sincronizar Empleados"><svg style="width:16px;height:16px;"><use href="#icon-refresh"></use></svg></button>
                        ${device.isDefault ? '' : `<button class="btn-action btn-action--icon btn-action--primary" onclick="setManagedDeviceDefault(decodeInlineArg('${encodeInlineArg(device.id || '')}'))" title="Establecer como predeterminado"><svg style="width:16px;height:16px;"><use href="#icon-check"></use></svg></button>`}
                        <button class="btn-action btn-action--icon btn-action--primary" onclick="editManagedDevice(decodeInlineArg('${encodeInlineArg(device.id || '')}'))" title="Editar"><svg style="width:16px;height:16px;"><use href="#icon-edit"></use></svg></button>
                        <button class="btn-action btn-action--icon btn-action--danger" onclick="deleteManagedDevice(decodeInlineArg('${encodeInlineArg(device.id || '')}'))" title="Eliminar"><svg style="width:16px;height:16px;"><use href="#icon-trash"></use></svg></button>
                    </div>
            </td>
        </tr>
    `).join('');
}

window.syncDeviceEmployees = async (id) => {
    const isDefault = id === 'default';
    const msg = isDefault ? '¿Sincronizar todos los empleados activos con el dispositivo predeterminado?' : '¿Sincronizar empleados con este dispositivo?';
    if (!confirm(msg)) return;

    showToast('Sincronizando empleados, por favor espere...');
    
    try {
        const resp = await fetch(`/api/devices/configured/${id}/sync`, {
            method: 'POST',
            headers: getAuthHeaders()
        });

        const result = await resp.json();

        if (resp.ok) {
            showToast(result.message || 'Sincronización exitosa');
        } else {
            showToast(result.error || 'Fallo en la sincronización', 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
}

window.syncDeviceTime = async (id) => {
    if (!confirm('¿Sincronizar la hora del dispositivo con la del servidor? Esto puede resolver errores de "Permiso Expirado".')) return;

    showToast('Sincronizando hora...');
    try {
        const resp = await fetch(`/api/devices/configured/${id}/sync-time`, {
            method: 'POST',
            headers: getAuthHeaders()
        });
        const result = await resp.json();
        if (resp.ok) {
            showToast(result.message || 'Hora sincronizada');
        } else {
            showToast(result.error || 'Error al sincronizar hora', 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
}

function openDeviceModal(device = null) {
    const modal = document.getElementById('device-modal');
    const form = document.getElementById('device-form');
    if (!modal || !form) return;

    form.reset();
    document.getElementById('device-id').value = device?.id || '';
    document.getElementById('device-name').value = device?.name || '';
    document.getElementById('device-ip').value = device?.ip || '';
    document.getElementById('device-port').value = device?.port || 80;
    document.getElementById('device-username').value = device?.username || '';
    document.getElementById('device-password').value = '';
    document.getElementById('device-source').value = device?.source || 'manual';
    document.getElementById('device-model').value = device?.model || '';
    document.getElementById('device-serial').value = device?.serial || '';
    document.getElementById('device-default').checked = !!device?.isDefault;
    document.getElementById('device-timezone').value = device?.timezoneOffset || '+08:00';
    document.getElementById('device-modal-title').innerText = device ? 'Editar Dispositivo' : 'Nuevo Dispositivo';
    modal.classList.add('active');
}

window.editManagedDevice = (id) => {
    const device = managedDevices.find(item => item.id === id);
    if (device) {
        openDeviceModal(device);
    }
};

window.createManagedDeviceFromScan = (ip, model, serial) => {
    openDeviceModal({
        name: model ? `${model} ${ip}` : `Terminal ${ip}`,
        ip,
        port: 80,
        username: 'admin',
        source: 'sadp',
        model,
        serial,
    });
};

window.deleteManagedDevice = async (id) => {
    if (!confirm('¿Eliminar este dispositivo del inventario administrado?')) return;

    try {
        const resp = await fetch(`/api/devices/configured/${id}`, {
            method: 'DELETE',
            headers: getAuthHeaders()
        });
        if (!resp.ok) {
            const err = await resp.text();
            showToast(`Error: ${err}`, 'error');
            return;
        }
        showToast('Dispositivo eliminado');
        await loadManagedDevices();
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
};

window.setManagedDeviceDefault = async (id) => {
    try {
        const resp = await fetch(`/api/devices/configured/${id}/default`, {
            method: 'POST',
            headers: getAuthHeaders()
        });
        if (!resp.ok) {
            const err = await resp.text();
            showToast(`Error: ${err}`, 'error');
            return;
        }
        showToast('Dispositivo predeterminado actualizado');
        await loadManagedDevices();
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
};

window.setupAlarmHost = async (id) => {
    if (!confirm('¿Configurar este dispositivo para enviar eventos en tiempo real al servidor?\n\nNota: Asegúrate de que la IP del servidor sea accesible desde el dispositivo.')) return;

    showToast('Configurando servidor de escucha (Alarm Host)...');
    
    try {
        const resp = await fetch(`/api/devices/configured/${id}/setup-alarm-host`, {
            method: 'POST',
            headers: getAuthHeaders()
        });

        if (resp.ok) {
            showToast('Dispositivo configurado exitosamente. Los eventos aparecerán en el dashboard en tiempo real.');
        } else {
            let errorMsg = 'Error desconocido';
            try {
                const data = await resp.json();
                errorMsg = data.error || data.message || errorMsg;
            } catch (e) {
                errorMsg = `Error del servidor (${resp.status})`;
            }
            showToast('Error al configurar: ' + errorMsg, 'error');
        }
        await loadManagedDevices();
    } catch (err) {
        console.error('Setup alarm host failed:', err);
        showToast('Error de comunicación con el servidor', 'error');
    }
};

// ==================== DEVICE ERROR MANAGER (HEALTH & LOGS) ====================

const DeviceErrorManager = {
    supportPhone: '18097649811',
    supportEmail: 'grupomv.rd@outlook.com',
    latestLogs: [],
    async fetchLogs() {
        try {
            const resp = await fetch('/api/devices/logs', {
                headers: getAuthHeaders()
            });

            if (!resp.ok) {
                console.error('Failed to fetch device logs:', await resp.text());
                return [];
            }

            const data = await resp.json();
            return Array.isArray(data) ? data : (data.logs || []);
        } catch (err) {
            console.error('Network error fetching device logs:', err);
            return [];
        }
    },

    renderLogs(logs) {
        const tbody = document.getElementById('device-logs-list');
        const countSpan = document.getElementById('health-error-count');
        const lastErrSpan = document.getElementById('health-last-failure');
        this.latestLogs = Array.isArray(logs) ? logs : [];

        if (!tbody) return;

        if (this.latestLogs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" style="text-align: center; color: var(--text-muted);">El sistema está funcionando correctamente. No hay errores recientes.</td></tr>';
            if (countSpan) countSpan.textContent = '0';
            if (lastErrSpan) lastErrSpan.textContent = 'N/D';
            return;
        }

        const activeErrors = this.latestLogs.filter(l => l.level === 'error').length;
        if (countSpan) countSpan.textContent = activeErrors;

        const lastErr = this.latestLogs.find(l => l.level === 'error');
        if (lastErrSpan && lastErr) {
            const date = new Date(lastErr.timestamp);
            lastErrSpan.textContent = `${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
        }

        tbody.innerHTML = this.latestLogs.map(log => {
            const levelClass = log.level === 'error' ? 'log-level-error' : (log.level === 'warning' ? 'log-level-warning' : 'log-level-info');
            const date = new Date(log.timestamp);
            const timeStr = `${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}`;
            
            // API returns camelCase: errorMessage, deviceId, operation, level, timestamp
            const rawMsg = log.errorMessage || log.error_message || '';
            let msg = escapeHTML(rawMsg);
            if (msg.includes('ISAPI error 400') || msg.includes('Invalid Operation')) {
                msg = `ISAPI 400: Operación Inválida`;
            }
            const devId = (log.deviceId || log.device_id || '').substring(0, 12);
            const opLabel = {
                'Sync': 'Sincronizar todo',
                'PushEmployee': 'Enviar empleado',
                'RevokeEmployee': 'Revocar acceso',
                'Connect': 'Conectar',
                'SyncTime': 'Sincronizar hora',
                'ReadEvents': 'Leer eventos',
                'RegisterFace': 'Registrar rostro',
                'DeleteFace': 'Eliminar rostro',
                'ImportFace': 'Importar rostro',
                'UploadPhoto': 'Subir foto'
            }[log.operation] || log.operation || '---';

            return `
                <tr>
                    <td class="log-time">${timeStr}</td>
                    <td>${escapeHTML(devId || 'N/D')}</td>
                    <td><span class="badge ${levelClass}">${escapeHTML((log.level || '').toUpperCase())}</span></td>
                    <td>${escapeHTML(opLabel)}</td>
                    <td class="log-message" title="${escapeHTML(rawMsg)}">${msg || '<span class="text-muted">OK</span>'}</td>
                </tr>
            `;
        }).join('');
    },

    buildReport(logs = this.latestLogs) {
        const items = (logs || []).slice(0, 20);
        const errors = items.filter(l => l.level === 'error').length;
        const warnings = items.filter(l => l.level === 'warning').length;
        const infos = items.filter(l => l.level === 'info').length;
        const lines = items.map(log => {
            const date = new Date(log.timestamp);
            const deviceId = log.deviceId || log.device_id || 'N/D';
            const message = log.errorMessage || log.error_message || 'OK';
            return `- ${date.toLocaleString()}: [${String(log.level || '').toUpperCase()}] ${log.operation || 'Operacion'} - ${message} (${deviceId})`;
        });
        return [
            'Reporte de Salud y Sync de Dispositivos',
            '',
            `Resumen: errores=${errors}, advertencias=${warnings}, info=${infos}`,
            '',
            'Ultimos registros:',
            ...lines
        ].join('\n');
    },

    async share(channel) {
        const logs = this.latestLogs.length ? this.latestLogs : await this.fetchLogs();
        this.renderLogs(logs);

        if (!logs.length) {
            showToast('No hay logs para enviar', 'error');
            return;
        }

        const body = this.buildReport(logs);
        if (channel === 'wa') {
            window.open(`https://wa.me/${this.supportPhone}?text=${encodeURIComponent(body)}`, '_blank', 'noopener');
            return;
        }

        const subject = 'Reporte de Salud y Sync de Dispositivos';
        window.location.href = `mailto:${encodeURIComponent(this.supportEmail)}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`;
    }
};

window.initDeviceErrorManager = () => {
    const btnRefresh = document.getElementById('btn-refresh-device-logs');
    const btnMail = document.getElementById('btn-send-device-logs-mail');
    const btnWA = document.getElementById('btn-send-device-logs-wa');
    if (btnRefresh) {
        btnRefresh.addEventListener('click', async () => {
            btnRefresh.disabled = true;
            btnRefresh.innerHTML = `<svg style="width:14px;height:14px;margin-right:4px;" class="spin"><use href="#icon-refresh"></use></svg> Refrescando...`;
            
            const logs = await DeviceErrorManager.fetchLogs();
            DeviceErrorManager.renderLogs(logs);
            
            btnRefresh.disabled = false;
            btnRefresh.innerHTML = `<svg style="width:14px;height:14px;margin-right:4px;"><use href="#icon-refresh"></use></svg> Refrescar Logs`;
        });
    }

    if (btnMail) {
        btnMail.addEventListener('click', () => DeviceErrorManager.share('mail'));
    }

    if (btnWA) {
        btnWA.addEventListener('click', () => DeviceErrorManager.share('wa'));
    }

    const tabBtns = document.querySelectorAll('.tab-btn');
    tabBtns.forEach(btn => {
        btn.addEventListener('click', async () => {
            if (btn.getAttribute('data-tab') === 'tab-company') {
                const logs = await DeviceErrorManager.fetchLogs();
                DeviceErrorManager.renderLogs(logs);
            }
        });
    });
};

// ==================== GRANT / REVOKE DEVICE ACCESS (SINGLE EMPLOYEE) ====================

// Grants access: sends employee to device terminal
window.grantDeviceAccess = async (employeeNo, employeeName) => {
    const defaultDevice = managedDevices.find(d => d.isDefault);

    if (managedDevices.length === 0) {
        showToast('No hay terminales configuradas. Agrega una en la sección Dispositivos.', 'error');
        return;
    }

    if (managedDevices.length === 1 || defaultDevice) {
        const targetId = defaultDevice ? defaultDevice.id : managedDevices[0].id;
        const targetName = defaultDevice ? defaultDevice.name : managedDevices[0].name;
        await _pushEmployeeToDevice(employeeNo, employeeName, targetId, targetName);
    } else {
        openDevicePickerModal(employeeNo, employeeName, 'grant');
    }
};

// Revokes access: removes employee from device WITHOUT deleting from DB
window.revokeDeviceAccess = async (employeeNo, employeeName) => {
    const defaultDevice = managedDevices.find(d => d.isDefault);

    if (managedDevices.length === 0) {
        showToast('No hay terminales configuradas.', 'error');
        return;
    }

    if (!confirm(`¿Revocar acceso de ${employeeName} en la terminal? El empleado NO será eliminado del sistema.`)) return;

    if (managedDevices.length === 1 || defaultDevice) {
        const targetId = defaultDevice ? defaultDevice.id : managedDevices[0].id;
        const targetName = defaultDevice ? defaultDevice.name : managedDevices[0].name;
        await _revokeEmployeeFromDevice(employeeNo, employeeName, targetId, targetName);
    } else {
        openDevicePickerModal(employeeNo, employeeName, 'revoke');
    }
};

async function _pushEmployeeToDevice(employeeNo, employeeName, deviceId, deviceName) {
    showToast(`Enviando ${employeeName} a ${deviceName}...`);
    try {
        const resp = await fetch(`/api/devices/configured/${deviceId}/sync-one/${encodeURIComponent(employeeNo)}`, {
            method: 'POST',
            headers: getAuthHeaders()
        });
        const result = await resp.json();
        if (resp.ok) {
            showToast(result.message || `✅ ${employeeName} registrado en ${deviceName}`);
            loadEmployees();
        } else {
            showToast(result.error || 'Error al registrar en el dispositivo', 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
}

async function _revokeEmployeeFromDevice(employeeNo, employeeName, deviceId, deviceName) {
    showToast(`Revocando acceso de ${employeeName} en ${deviceName}...`);
    try {
        const resp = await fetch(`/api/devices/configured/${deviceId}/sync-one/${encodeURIComponent(employeeNo)}`, {
            method: 'DELETE',
            headers: getAuthHeaders()
        });
        const result = await resp.json();
        if (resp.ok) {
            showToast(result.message || `✅ Acceso de ${employeeName} revocado en ${deviceName}`);
            loadEmployees();
        } else {
            showToast(result.error || 'Error al revocar acceso en el dispositivo', 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
}

// action: 'grant' | 'revoke'
function openDevicePickerModal(employeeNo, employeeName, action = 'grant') {
    const modal = document.getElementById('device-picker-modal');
    if (!modal) return;

    document.getElementById('device-picker-emp-name').textContent = employeeName;
    document.getElementById('device-picker-emp-no').value = employeeNo;
    modal.dataset.action = action;

    const title = modal.querySelector('.modal-title') || modal.querySelector('h3');
    if (title) title.textContent = action === 'revoke' ? 'Revocar Acceso - Seleccionar Terminal' : 'Otorgar Acceso - Seleccionar Terminal';

    const list = document.getElementById('device-picker-list');
    list.innerHTML = managedDevices.map(d => `
        <button class="device-picker-item ${d.isDefault ? 'device-picker-item--default' : ''}"
            onclick="pickDeviceForEmployee(decodeInlineArg('${encodeInlineArg(d.id)}'), decodeInlineArg('${encodeInlineArg(d.name)}'))">
            <span class="device-picker-icon">🖥️</span>
            <span class="device-picker-info">
                <strong>${escapeHTML(d.name)}</strong>
                <small>${escapeHTML(d.ip)}:${d.port || 80} ${d.isDefault ? '<em>(predeterminado)</em>' : ''}</small>
            </span>
        </button>
    `).join('');

    modal.classList.add('active');
}

window.pickDeviceForEmployee = async (deviceId, deviceName) => {
    const modal = document.getElementById('device-picker-modal');
    const employeeNo = document.getElementById('device-picker-emp-no').value;
    const employeeName = document.getElementById('device-picker-emp-name').textContent;
    const action = modal.dataset.action || 'grant';
    modal.classList.remove('active');
    if (action === 'revoke') {
        await _revokeEmployeeFromDevice(employeeNo, employeeName, deviceId, deviceName);
    } else {
        await _pushEmployeeToDevice(employeeNo, employeeName, deviceId, deviceName);
    }
};

window.allEmployeesData = [];
window.deptMapCache = {};

async function loadEmployees() {
    try {
        const [empResp, deptResp] = await Promise.all([
            fetch('/api/employees', { headers: getAuthHeaders() }),
            fetch('/api/departments', { headers: getAuthHeaders() })
        ]);

        if (!empResp.ok) {
            const errText = await empResp.text();
            console.error('API /api/employees error:', errText);
            showToast('Error al cargar empleados: ' + errText, 'error');
            return;
        }

        const emps = await empResp.json();
        const depts = await deptResp.json();

        // Create department map
        window.deptMapCache = {};
        if (depts && Array.isArray(depts)) {
            depts.forEach(d => { window.deptMapCache[d.id] = d.name; });
        }

        window.allEmployeesData = emps || [];
        console.log('Employees loaded:', window.allEmployeesData.length);
        renderEmployeesTable(window.allEmployeesData);
    } catch (err) {
        console.error('Load employees failed exception:', err);
        showToast('Error de conexión al cargar empleados', 'error');
    }
}

function renderEmployeesTable(employees) {
    const list = document.getElementById('employees-list');
    if (!list) return;

    if (!employees || employees.length === 0) {
        list.innerHTML = '<tr><td colspan="5" class="text-muted" style="text-align: center; padding: 3rem;">No hay empleados encontrados.</td></tr>';
        document.getElementById('employees-row-count').textContent = 'Mostrando 0 de 0 registros';
        document.getElementById('employees-pagination').innerHTML = '';
        return;
    }

    const canEditEmployees = canManageOrganization();
    const canManageFaces = canManageOrganization();
    const canGrantAccess = canManageDevices();
    
    // Paginate
    const totalPages = Math.ceil(employees.length / PAGINATION_SIZE);
    if (employeesPage > totalPages) employeesPage = totalPages || 1;
    const start = (employeesPage - 1) * PAGINATION_SIZE;
    const end = start + PAGINATION_SIZE;
    const pageItems = employees.slice(start, end);
    
    list.innerHTML = pageItems.map(e => {
        const deptName = window.deptMapCache[e.departmentId] || '---';
        const badgeClass = e.status === 'Active' ? 'badge-success' : 'badge-secondary';
        const fullName = `${e.firstName || ''} ${e.lastName || ''}`.trim();
        const photoSrc = employeePhotoSrc(e);
        
        return `
        <tr>
            <td><strong>${escapeHTML(e.employeeNo || '')}</strong></td>
            <td>
                <div style="display:flex; align-items:center; gap:10px;">
                    <div style="width:32px;height:32px;border-radius:50%;background:rgba(255,255,255,0.05);display:flex;align-items:center;justify-content:center;font-size:14px;overflow:hidden;border:1px solid var(--border-strong);">
                        ${photoSrc ? `<img src="${photoSrc}" style="width:100%;height:100%;object-fit:cover;">` : '👤'}
                    </div>
                    <div style="display:flex; flex-direction:column; gap:2px;">
                        <span style="font-weight:500;">${escapeHTML(fullName)}</span>
                        ${e.isSystemAdmin ? '<span class="badge badge-danger" style="font-size:0.62rem; width:max-content;">Admin del sistema</span>' : ''}
                    </div>
                </div>
            </td>
            <td>${escapeHTML(deptName)}</td>
            <td><span class="badge ${badgeClass}">${escapeHTML(e.status || '')}</span></td>
            <td>
                <div class="travel-actions">
                    ${canGrantAccess ? `
                        <button class="btn-action btn-action--icon btn-action--view" title="Registrar en terminal" onclick="grantDeviceAccess(decodeInlineArg('${encodeInlineArg(e.employeeNo || '')}'), decodeInlineArg('${encodeInlineArg(fullName)}'))">
                            <svg><use href="#icon-key"></use></svg>
                        </button>
                        <button class="btn-action btn-action--icon btn-action--danger" title="Revocar acceso en terminal" onclick="revokeDeviceAccess(decodeInlineArg('${encodeInlineArg(e.employeeNo || '')}'), decodeInlineArg('${encodeInlineArg(fullName)}'))">
                            <svg><use href="#icon-x"></use></svg>
                        </button>` : ''}
                    ${canManageFaces ? `
                        <button class="btn-action btn-action--icon btn-action--view" title="Gestionar registro facial" onclick="openFaceModal(decodeInlineArg('${encodeInlineArg(e.employeeNo || '')}'))">
                            <svg><use href="#icon-camera"></use></svg>
                        </button>` : ''}
                    ${canEditEmployees ? `
                        <button class="btn-action btn-action--icon btn-action--primary" title="Editar empleado" onclick="editEmployee(decodeInlineArg('${encodeInlineArg(e.id || '')}'))">
                            <svg><use href="#icon-edit"></use></svg>
                        </button>` : ''}
                </div>
            </td>
        </tr>
        `;
    }).join('');

    renderCommonPagination('employees-pagination', employees.length, employeesPage, 'changeEmployeesPage', 'employees-row-count');
}

window.changeEmployeesPage = (p) => {
    employeesPage = p;
    const term = document.getElementById('employee-search')?.value.toLowerCase() || '';
    const filtered = (window.allEmployeesData || []).filter(emp => {
        const fullName = `${emp.firstName || ''} ${emp.lastName || ''}`.toLowerCase();
        return fullName.includes(term) || (emp.employeeNo && emp.employeeNo.toLowerCase().includes(term));
    });
    renderEmployeesTable(filtered);
    document.querySelector('#employees .users-table-container').scrollTop = 0;
};

// Client-side search listener
document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('employee-search');
    if (searchInput) {
        searchInput.addEventListener('input', (e) => {
            employeesPage = 1; // Reset to page 1 on search
            const term = e.target.value.toLowerCase();
            const filtered = (window.allEmployeesData || []).filter(emp => {
                const fullName = `${emp.firstName || ''} ${emp.lastName || ''}`.toLowerCase();
                return fullName.includes(term) || (emp.employeeNo && emp.employeeNo.toLowerCase().includes(term));
            });
            renderEmployeesTable(filtered);
        });
    }
});

async function loadEmployeeDepartments() {
    try {
        const resp = await fetch('/api/departments', {
            headers: getAuthHeaders()
        });
        const depts = await resp.json();
        const select = document.getElementById('select-dept');
        if (!select) return;
        
        select.innerHTML = '<option value="">Seleccionar Departamento</option>' + 
            depts.map(d => `<option value="${escapeHTML(d.id || '')}">${escapeHTML(d.name || '')}</option>`).join('');
    } catch (err) {
        console.error('Failed to load departments', err);
    }
}

async function loadEmployeePositions() {
    try {
        const resp = await fetch('/api/positions', {
            headers: getAuthHeaders()
        });
        const positions = await resp.json();
        const select = document.getElementById('select-pos');
        if (!select) return;

        select.innerHTML = '<option value="">Seleccionar Cargo</option>' +
            positions.map(p => `<option value="${escapeHTML(p.id || '')}">${escapeHTML(p.name || '')}</option>`).join('');
    } catch (err) {
        console.error('Failed to load positions', err);
    }
}

// ==================== TABS ====================

function initTabs() {
    const tabBtns = document.querySelectorAll('.tab-btn');

    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const tabId = btn.getAttribute('data-tab');

            // Remove active class from all tabs and contents
            tabBtns.forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));

            // Add active class to selected tab
            btn.classList.add('active');
            document.getElementById(tabId).classList.add('active');

            // Load users when Users tab is activated
            if (tabId === 'tab-users') {
                loadUsers();
                loadLDAPUsers();
            }
        });
    });
}

// ==================== USERS MANAGEMENT ====================

function initUsers() {
    const modal = document.getElementById('user-modal');
    const btnAdd = document.getElementById('btn-add-user');
    const btnRefresh = document.getElementById('btn-refresh-ldap');
    const closeBtns = modal.querySelectorAll('.close-modal');
    const form = document.getElementById('user-form');

    // Open modal for new user
    if (btnAdd) {
        btnAdd.addEventListener('click', () => {
            document.getElementById('user-modal-title').innerText = 'Nuevo Usuario';
            document.getElementById('user-form').reset();
            document.getElementById('user-id').value = '';
            document.getElementById('user-employee-id').value = '';
            document.getElementById('user-password').required = true;
            modal.classList.add('active');
        });
    }

    // Close modal
    closeBtns.forEach(btn => {
        btn.addEventListener('click', () => modal.classList.remove('active'));
    });

    // Form submit
    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());

        const userId = data.id;
        delete data.id;
        delete data.employeeId;

        // Password is optional when editing
        if (!data.password && userId) {
            delete data.password;
        }

        try {
            const url = userId ? `/api/users/${userId}` : '/api/users';
            const method = userId ? 'PUT' : 'POST';

            const resp = await fetch(url, {
                method: method,
                headers: getAuthHeaders(),
                body: JSON.stringify(data)
            });

            if (resp.ok) {
                showToast(userId ? 'Usuario actualizado' : 'Usuario creado');
                modal.classList.remove('active');
                loadUsers();
            } else {
                const err = await resp.text();
                showToast(`Error: ${err}`, 'error');
            }
        } catch (err) {
            showToast('Error de conexión', 'error');
        }
    });

    // Refresh LDAP users
    if (btnRefresh) {
        btnRefresh.addEventListener('click', loadLDAPUsers);
    }
}

async function loadUsers() {
    try {
        const resp = await fetch('/api/users', {
            headers: getAuthHeaders()
        });
        const users = await resp.json();
        const tbody = document.getElementById('users-list');
        if (!tbody) return;

        if (!users || users.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="text-muted" style="text-align: center; padding: 2rem;">No hay usuarios registrados</td></tr>';
            return;
        }

        tbody.innerHTML = users.map(u => `
            <tr>
                <td>${escapeHTML(u.username || '')}</td>
                <td>${escapeHTML(u.fullName || '---')}</td>
                <td>${escapeHTML(u.email || '---')}</td>
                <td><span class="badge ${u.role === 'admin' ? 'badge-danger' : u.role === 'manager' ? 'badge-warning' : 'badge-secondary'}">${escapeHTML(u.role || '')}</span></td>
                <td>
                    <div class="travel-actions">
                        <button class="btn-action btn-action--icon btn-action--primary" onclick="editUser(decodeInlineArg('${encodeInlineArg(u.id || '')}'), decodeInlineArg('${encodeInlineArg(u.username || '')}'), decodeInlineArg('${encodeInlineArg(u.fullName || '')}'), decodeInlineArg('${encodeInlineArg(u.email || '')}'), decodeInlineArg('${encodeInlineArg(u.role || '')}'))" title="Editar"><svg style="width:16px;height:16px;"><use href="#icon-edit"></use></svg></button>
                        ${u.username !== 'admin' ? `<button class="btn-action btn-action--icon btn-action--danger" onclick="deleteUser(decodeInlineArg('${encodeInlineArg(u.id || '')}'))" title="Eliminar"><svg style="width:16px;height:16px;"><use href="#icon-trash"></use></svg></button>` : ''}
                    </div>
                </td>
            </tr>
        `).join('');
    } catch (err) {
        console.error('Load users failed', err);
    }
}

async function loadLDAPUsers() {
    try {
        const resp = await fetch('/api/employees', {
            headers: getAuthHeaders()
        });
        const employees = await resp.json();

        // Get existing users to filter out
        const usersResp = await fetch('/api/users', {
            headers: getAuthHeaders()
        });
        const users = await usersResp.json();
        const usersEmails = new Set(users.map(u => u.email?.toLowerCase()));

        const tbody = document.getElementById('ldap-users-list');
        if (!tbody) return;

        // Filter employees without users (by email match)
        const ldapUsers = employees.filter(e => e.email && !usersEmails.has(e.email.toLowerCase()));

        if (ldapUsers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="text-muted" style="text-align: center; padding: 2rem;">No hay empleados LDAP pendientes de crear usuario</td></tr>';
            return;
        }

        tbody.innerHTML = ldapUsers.map(e => `
            <tr>
                <td>${escapeHTML(e.employeeNo || '')}</td>
                <td>${escapeHTML(`${e.firstName || ''} ${e.lastName || ''}`.trim())}</td>
                <td>${escapeHTML(e.departmentId || '---')}</td>
                <td>${escapeHTML(e.email || '---')}</td>
                <td>
                    <button class="btn btn-sm" onclick="createUserFromEmployee(decodeInlineArg('${encodeInlineArg(e.id || '')}'), decodeInlineArg('${encodeInlineArg(e.firstName || '')}'), decodeInlineArg('${encodeInlineArg(e.lastName || '')}'), decodeInlineArg('${encodeInlineArg(e.email || '')}'))">Crear Usuario</button>
                </td>
            </tr>
        `).join('');
    } catch (err) {
        console.error('Load LDAP users failed', err);
    }
}

window.editUser = (id, username, fullName, email, role) => {
    document.getElementById('user-modal-title').innerText = 'Editar Usuario';
    document.getElementById('user-id').value = id;
    document.getElementById('user-username').value = username;
    document.getElementById('user-fullname').value = fullName;
    document.getElementById('user-email').value = email;
    document.getElementById('user-role').value = role;
    document.getElementById('user-password').value = '';
    document.getElementById('user-password').required = false;
    document.getElementById('user-modal').classList.add('active');
};

window.createUserFromEmployee = (employeeId, firstName, lastName, email) => {
    document.getElementById('user-modal-title').innerText = 'Crear Usuario desde LDAP';
    document.getElementById('user-employee-id').value = employeeId;
    document.getElementById('user-username').value = email ? email.split('@')[0] : '';
    document.getElementById('user-fullname').value = `${firstName} ${lastName}`;
    document.getElementById('user-email').value = email;
    document.getElementById('user-role').value = 'viewer';
    document.getElementById('user-password').value = '';
    document.getElementById('user-password').required = true;
    document.getElementById('user-modal').classList.add('active');
};

window.deleteUser = async (id) => {
    if (!confirm('¿Estás seguro de eliminar este usuario?')) return;

    try {
        const resp = await fetch(`/api/users/${id}`, {
            method: 'DELETE',
            headers: getAuthHeaders()
        });

        if (resp.ok) {
            showToast('Usuario eliminado');
            loadUsers();
        } else {
            const err = await resp.text();
            showToast(`Error: ${err}`, 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
};

// ==================== DEPARTMENTS ====================

function initDepartments() {
    const modal = document.getElementById('dept-modal');
    const btnAdd = document.getElementById('btn-add-dept');
    const closeBtns = modal.querySelectorAll('.close-modal');
    const form = document.getElementById('dept-form');

    if (btnAdd) {
        btnAdd.addEventListener('click', async () => {
            document.getElementById('dept-modal-title').innerText = 'Nuevo Departamento';
            document.getElementById('dept-form').reset();
            document.getElementById('dept-id').value = '';
            
            // Show modal immediately
            modal.classList.add('active');
            
            try {
                await loadEmployeesForDeptSelect();
            } catch(e) {
                console.error('Error loading employees for department select', e);
            }
        });
    }

    closeBtns.forEach(btn => {
        btn.addEventListener('click', () => modal.classList.remove('active'));
    });

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());

        const deptId = data.id;
        delete data.id;

        try {
            const url = deptId ? `/api/departments/${deptId}` : '/api/departments';
            const method = deptId ? 'PUT' : 'POST';

            const resp = await fetch(url, {
                method: method,
                headers: getAuthHeaders(),
                body: JSON.stringify(data)
            });

            if (resp.ok) {
                showToast(deptId ? 'Departamento actualizado' : 'Departamento creado');
                modal.classList.remove('active');
                loadDepartments();
            } else {
                const err = await resp.text();
                showToast(`Error: ${err}`, 'error');
            }
        } catch (err) {
            showToast('Error de conexión', 'error');
        }
    });
}

async function loadDepartments() {
    try {
        const resp = await fetch('/api/departments', { headers: getAuthHeaders() });
        const depts = await resp.json();
        window.allDepartmentsData = depts || [];
        renderDepartmentsTable(window.allDepartmentsData);
    } catch (err) {
        showToast('Error de conexión al cargar departamentos', 'error');
    }
}

async function renderDepartmentsTable(depts) {
    const tbody = document.getElementById('departments-list');
    if (!tbody) return;

    if (!depts || depts.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-muted" style="text-align: center; padding: 2rem;">No hay departamentos registrados</td></tr>';
        document.getElementById('departments-row-count').textContent = 'Mostrando 0 de 0 registros';
        document.getElementById('departments-pagination').innerHTML = '';
        return;
    }

    // Count employees per department (cache this if performance is an issue)
    const empResp = await fetch('/api/employees', { headers: getAuthHeaders() });
    const employees = await empResp.json();
    const empCount = {};
    employees.forEach(e => {
        if (e.departmentId) empCount[e.departmentId] = (empCount[e.departmentId] || 0) + 1;
    });

    const totalPages = Math.ceil(depts.length / PAGINATION_SIZE);
    if (departmentsPage > totalPages) departmentsPage = totalPages || 1;
    const start = (departmentsPage - 1) * PAGINATION_SIZE;
    const end = start + PAGINATION_SIZE;
    const pageItems = depts.slice(start, end);

    const canEditDepartments = canManageOrganization();
    tbody.innerHTML = pageItems.map(d => `
        <tr>
            <td>
                <strong>${escapeHTML(d.name || '')}</strong><br>
                <small class="text-muted">Encargado: ${escapeHTML(d.managerName || 'Sin asignar')}</small>
            </td>
            <td>${escapeHTML(d.description || '---')}</td>
            <td>${empCount[d.id] || 0} empleados</td>
            <td class="travel-actions-cell">
                <div class="travel-actions">
                    ${canEditDepartments ? `<button class="btn-action btn-action--icon btn-action--primary" onclick="editDept(decodeInlineArg('${encodeInlineArg(d.id || '')}'), decodeInlineArg('${encodeInlineArg(d.name || '')}'), decodeInlineArg('${encodeInlineArg(d.description || '')}'), decodeInlineArg('${encodeInlineArg(d.managerId || '')}'))" title="Editar"><svg style="width:16px;height:16px;"><use href="#icon-edit"></use></svg></button>` : '<span class="text-muted">Solo lectura</span>'}
                    ${canEditDepartments ? `<button class="btn-action btn-action--icon btn-action--danger" onclick="deleteDept(decodeInlineArg('${encodeInlineArg(d.id || '')}'))" title="Eliminar"><svg style="width:16px;height:16px;"><use href="#icon-trash"></use></svg></button>` : ''}
                </div>
            </td>
        </tr>
    `).join('');

    renderCommonPagination('departments-pagination', depts.length, departmentsPage, 'changeDepartmentsPage', 'departments-row-count');
}

window.changeDepartmentsPage = (p) => {
    departmentsPage = p;
    renderDepartmentsTable(window.allDepartmentsData);
    document.querySelector('#departments .users-table-container').scrollTop = 0;
};

async function loadEmployeesForDeptSelect() {
    try {
        const resp = await fetch('/api/employees', { headers: getAuthHeaders() });
        const emps = await resp.json();
        const select = document.getElementById('dept-manager');
        if (!select) return;
        select.innerHTML = '<option value="">Ninguno / Seleccionar después</option>' + 
            (emps || []).map(e => `<option value="${escapeHTML(e.id || '')}">${escapeHTML(`${e.firstName || ''} ${e.lastName || ''}`.trim())}</option>`).join('');
    } catch (err) {
        console.error('Load employees failed', err);
    }
}

window.editDept = async (id, name, description, managerId) => {
    document.getElementById('dept-modal-title').innerText = 'Editar Departamento';
    document.getElementById('dept-id').value = id;
    document.getElementById('dept-name').value = name;
    document.getElementById('dept-description').value = description;
    await loadEmployeesForDeptSelect();
    document.getElementById('dept-manager').value = managerId || '';
    document.getElementById('dept-modal').classList.add('active');
};

window.deleteDept = async (id) => {
    if (!confirm('¿Estás seguro de eliminar este departamento?')) return;

    try {
        const resp = await fetch(`/api/departments/${id}`, {
            method: 'DELETE',
            headers: getAuthHeaders()
        });

        if (resp.ok) {
            showToast('Departamento eliminado');
            loadDepartments();
        } else {
            const err = await resp.text();
            showToast(`Error: ${err}`, 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
};

// ==================== POSITIONS ====================

function initPositions() {
    const modal = document.getElementById('pos-modal');
    const btnAdd = document.getElementById('btn-add-pos');
    const closeBtns = modal.querySelectorAll('.close-modal');
    const form = document.getElementById('pos-form');

    if (btnAdd) {
        btnAdd.addEventListener('click', async () => {
            document.getElementById('pos-modal-title').innerText = 'Nuevo Cargo';
            document.getElementById('pos-form').reset();
            document.getElementById('pos-id').value = '';
            document.getElementById('pos-level').value = '1';
            await loadDepartmentsForSelect();
            modal.classList.add('active');
        });
    }

    closeBtns.forEach(btn => {
        btn.addEventListener('click', () => modal.classList.remove('active'));
    });

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());

        const posId = data.id;
        delete data.id;
        data.level = parseInt(data.level) || 1;

        try {
            const url = posId ? `/api/positions/${posId}` : '/api/positions';
            const method = posId ? 'PUT' : 'POST';

            const resp = await fetch(url, {
                method: method,
                headers: getAuthHeaders(),
                body: JSON.stringify(data)
            });

            if (resp.ok) {
                showToast(posId ? 'Cargo actualizado' : 'Cargo creado');
                modal.classList.remove('active');
                loadPositions();
            } else {
                const err = await resp.text();
                showToast(`Error: ${err}`, 'error');
            }
        } catch (err) {
            showToast('Error de conexión', 'error');
        }
    });
}

async function loadDepartmentsForSelect() {
    try {
        const resp = await fetch('/api/departments', {
            headers: getAuthHeaders()
        });
        const depts = await resp.json();
        const select = document.getElementById('pos-dept-select');
        if (!select) return;

        select.innerHTML = '<option value="">Seleccionar Departamento</option>' +
            depts.map(d => `<option value="${escapeHTML(d.id || '')}">${escapeHTML(d.name || '')}</option>`).join('');
    } catch (err) {
        console.error('Failed to load departments for select', err);
    }
}

async function loadPositions() {
    try {
        const resp = await fetch('/api/positions', { headers: getAuthHeaders() });
        const positions = await resp.json();
        window.allPositionsData = positions || [];
        renderPositionsTable(window.allPositionsData);
    } catch (err) {
        showToast('Error de conexión al cargar cargos', 'error');
    }
}

async function renderPositionsTable(positions) {
    const tbody = document.getElementById('positions-list');
    if (!tbody) return;

    if (!positions || positions.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-muted" style="text-align: center; padding: 2rem;">No hay cargos registrados</td></tr>';
        document.getElementById('positions-row-count').textContent = 'Mostrando 0 de 0 registros';
        document.getElementById('positions-pagination').innerHTML = '';
        return;
    }

    const totalPages = Math.ceil(positions.length / PAGINATION_SIZE);
    if (positionsPage > totalPages) positionsPage = totalPages || 1;
    const start = (positionsPage - 1) * PAGINATION_SIZE;
    const end = start + PAGINATION_SIZE;
    const pageItems = positions.slice(start, end);

    const canEditPositions = canManageOrganization();
    tbody.innerHTML = pageItems.map(p => `
        <tr>
            <td><strong>${escapeHTML(p.name || '')}</strong></td>
            <td>${escapeHTML(window.deptMapCache[p.departmentId] || '---')}</td>
            <td>Nivel ${p.level || 0}</td>
            <td>${(window.allEmployeesData || []).filter(e => e.positionId === p.id).length} empleados</td>
            <td class="travel-actions-cell">
                <div class="travel-actions">
                    ${canEditPositions ? `
                        <button class="btn-action btn-action--icon btn-action--primary" onclick="editPosition(decodeInlineArg('${encodeInlineArg(p.id || '')}'), decodeInlineArg('${encodeInlineArg(p.name || '')}'), decodeInlineArg('${encodeInlineArg(p.departmentId || '')}'), ${p.level || 0})" title="Editar"><svg style="width:16px;height:16px;"><use href="#icon-edit"></use></svg></button>
                        <button class="btn-action btn-action--icon btn-action--danger" onclick="deletePosition(decodeInlineArg('${encodeInlineArg(p.id || '')}'))" title="Eliminar"><svg style="width:16px;height:16px;"><use href="#icon-trash"></use></svg></button>
                    ` : '<span class="text-muted">Solo lectura</span>'}
                </div>
            </td>
        </tr>
    `).join('');

    renderCommonPagination('positions-pagination', positions.length, positionsPage, 'changePositionsPage', 'positions-row-count');
}

window.changePositionsPage = (p) => {
    positionsPage = p;
    renderPositionsTable(window.allPositionsData);
    document.querySelector('#positions .users-table-container').scrollTop = 0;
};

window.editPos = async (id, name, departmentId, level) => {
    document.getElementById('pos-modal-title').innerText = 'Editar Cargo';
    document.getElementById('pos-id').value = id;
    document.getElementById('pos-name').value = name;
    document.getElementById('pos-level').value = level;
    await loadDepartmentsForSelect();
    if (departmentId) {
        document.getElementById('pos-dept-select').value = departmentId;
    }
    document.getElementById('pos-modal').classList.add('active');
};

window.deletePos = async (id) => {
    if (!confirm('¿Estás seguro de eliminar este cargo?')) return;

    try {
        const resp = await fetch(`/api/positions/${id}`, {
            method: 'DELETE',
            headers: getAuthHeaders()
        });

        if (resp.ok) {
            showToast('Cargo eliminado');
            loadPositions();
        } else {
            const err = await resp.text();
            showToast(`Error: ${err}`, 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
};


// ==================== AUDIT LOG MANAGER ====================

const AuditLogManager = {
    async fetchLogs() {
        try {
            const resp = await fetch('/api/audit-logs', {
                headers: getAuthHeaders()
            });
            if (!resp.ok) return [];
            return await resp.json();
        } catch (err) {
            console.error('Fetch audit logs failed', err);
            return [];
        }
    },

    async render() {
        const tbody = document.getElementById('audit-list');
        if (!tbody) return;

        tbody.innerHTML = '<tr><td colspan="6" style="text-align:center; padding:2rem;">Cargando registros...</td></tr>';
        
        const logs = await this.fetchLogs();
        
        if (!logs || logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center; padding:2rem; color:var(--text-muted);">No hay registros de auditoría disponibles.</td></tr>';
            return;
        }

        tbody.innerHTML = logs.map(log => {
            const date = new Date(log.timestamp);
            const dateStr = `${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}`;
            return `
                <tr>
                    <td><span class="text-muted" style="font-size:0.8rem;">${dateStr}</span></td>
                    <td><span class="badge badge-info">${escapeHTML(log.action)}</span></td>
                    <td><span class="badge badge-secondary">${escapeHTML(log.resource)}</span></td>
                    <td><div style="max-width:300px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title="${escapeHTML(log.details || '')}">${escapeHTML(log.details || '')}</div></td>
                    <td><strong>${escapeHTML(log.username || 'Sistema')}</strong></td>
                    <td><code style="font-size:0.75rem;">${escapeHTML(log.ipAddress || '')}</code></td>
                </tr>
            `;
        }).join('');
    }
};

window.refreshAuditLogs = () => AuditLogManager.render();

// ==================== ATTENDANCE ====================



// ==================== VIATICOS ====================

let travelRatesCache = [];   // cached for calc preview
let travelEmpsCache = [];    // cached for calc preview
let travelCurrentFilter = 'all';
let travelDecisionId = null;
let travelDecisionType = null;

function getTravelSelectedEmployeeIds() {
    const requestType = document.getElementById('travel-request-type')?.value || 'single';
    if (requestType === 'group') {
        const items = document.querySelectorAll('#travel-selected-list .transfer-item');
        return Array.from(items).map(item => item.dataset.id).filter(Boolean);
    }
    const single = document.getElementById('travel-employee')?.value;
    return single ? [single] : [];
}

function updateTravelParticipantsCount() {
    const count = getTravelSelectedEmployeeIds().length;
    const el = document.getElementById('travel-participants-count');
    if (el) el.innerText = `${count} seleccionado${count !== 1 ? 's' : ''}`;
}

function renderTravelParticipants(employees, selectedIds = [], filter = '') {
    const availableList = document.getElementById('travel-available-list');
    const selectedList = document.getElementById('travel-selected-list');
    if (!availableList || !selectedList) return;

    const term = filter.toLowerCase().trim();
    
    // Split employees
    const selected = employees.filter(e => selectedIds.includes(e.id));
    const available = employees.filter(e => !selectedIds.includes(e.id));

    // Render Available
    const filteredAvailable = available.filter(e => {
        if (!term) return true;
        return `${e.firstName || ''} ${e.lastName || ''} ${e.employeeNo || ''}`.toLowerCase().includes(term);
    });

    if (filteredAvailable.length === 0) {
        availableList.innerHTML = `<div class="text-muted" style="text-align:center; padding: 20px; font-size:0.8rem;">${term ? 'No hay coincidencias' : 'Todos seleccionados'}</div>`;
    } else {
        availableList.innerHTML = filteredAvailable.map(e => `
            <div class="transfer-item" data-id="${e.id}" onclick="moveParticipant('${e.id}', true)">
                <div class="info">
                    <div class="name">${escapeHTML(`${e.firstName} ${e.lastName}`)}</div>
                    <div class="meta">#${escapeHTML(e.employeeNo)}</div>
                </div>
                <div class="icon">
                    <svg style="width:16px;height:16px;"><use href="#icon-plus"></use></svg>
                </div>
            </div>
        `).join('');
    }

    // Render Selected
    if (selected.length === 0) {
        selectedList.innerHTML = `<div class="text-muted" style="text-align:center; padding: 20px; font-size:0.8rem;">Nadie seleccionado aún</div>`;
    } else {
        selectedList.innerHTML = selected.map(e => `
            <div class="transfer-item transfer-item--selected" data-id="${e.id}" onclick="moveParticipant('${e.id}', false)">
                <div class="info">
                    <div class="name">${escapeHTML(`${e.firstName} ${e.lastName}`)}</div>
                    <div class="meta">#${escapeHTML(e.employeeNo)}</div>
                </div>
                <div class="icon">
                    <svg style="width:16px;height:16px;"><use href="#icon-trash"></use></svg>
                </div>
            </div>
        `).join('');
    }

    const countEl = document.getElementById('travel-participants-count');
    if (countEl) countEl.innerText = selected.length;
    updateTravelCalcPreview();
}

window.moveParticipant = (id, toSelected) => {
    const currentSelected = getTravelSelectedEmployeeIds();
    let newSelected;
    if (toSelected) {
        newSelected = [...new Set([...currentSelected, id])];
    } else {
        newSelected = currentSelected.filter(sid => sid !== id);
    }
    
    const searchTerm = document.getElementById('travel-participants-search')?.value || '';
    renderTravelParticipants(travelEmpsCache, newSelected, searchTerm);
};

// Restoring missing travel functions
window.viewTravelDetails = async (id) => {
    const ta = (window.allTravelData || []).find(t => t.id === id);
    if (!ta) {
        showToast('Solicitud no encontrada', 'error');
        return;
    }
    if (canManageOrganization()) {
        openTravelModal(id);
    } else {
        showToast('Destino: ' + ta.destination + ' - Estado: ' + ta.status, 'info');
    }
};

window.decideTravel = async (id, status) => {
    const ta = (window.allTravelData || []).find(t => t.id === id);
    if (!ta) return;

    const action = status === 'Approved' ? 'Aprobar' : 'Rechazar';
    if (!confirm(`¿Estás seguro de que deseas ${action.toLowerCase()} esta solicitud?`)) return;

    try {
        const resp = await fetch(`/api/travel-allowances/${id}/status`, {
            method: 'PUT',
            headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
            body: JSON.stringify({ status })
        });

        if (resp.ok) {
            showToast(`Solicitud ${status === 'Approved' ? 'aprobada' : 'rechazada'}`);
            loadTravelAllowances();
        } else {
            const err = await resp.json();
            showToast(err.error || 'Error al procesar decisión', 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
};

function setTravelRequestTypeUI() {
    const requestType = document.getElementById('travel-request-type')?.value || 'single';
    const singleWrap = document.getElementById('travel-employee')?.closest('.form-group');
    const groupWrap = document.getElementById('travel-group-wrap');
    const groupName = document.getElementById('travel-group-name');
    const modal = document.getElementById('travel-modal')?.querySelector('.modal-card');

    if (singleWrap) singleWrap.style.display = requestType === 'single' ? 'block' : 'none';
    if (groupWrap) groupWrap.style.display = requestType === 'group' ? 'block' : 'none';
    if (groupName) groupName.disabled = requestType !== 'group';
    
    if (modal) {
        if (requestType === 'group') {
            modal.classList.add('wide-mode');
        } else {
            modal.classList.remove('wide-mode');
        }
    }
    
    updateTravelCalcPreview();
}

function initTravelAllowances() {
    // --- Modal helpers ---
    document.querySelectorAll('.close-modal[data-modal]').forEach(btn => {
        btn.addEventListener('click', () => {
            const id = btn.dataset.modal;
            const el = document.getElementById(id);
            if (el) el.classList.remove('active');
        });
    });

    // --- Nueva solicitud ---
    const btnNew = document.getElementById('btn-new-travel');
    if (btnNew) btnNew.addEventListener('click', () => openTravelModal());

    // --- Gestión de tarifas ---
    const btnRates = document.getElementById('btn-manage-rates');
    if (btnRates) btnRates.addEventListener('click', () => openRatesModal());

    const requestType = document.getElementById('travel-request-type');
    if (requestType) requestType.addEventListener('change', setTravelRequestTypeUI);

    const pSearch = document.getElementById('travel-participants-search');
    if (pSearch) {
        pSearch.addEventListener('input', (e) => {
            renderTravelParticipants(travelEmpsCache, getTravelSelectedEmployeeIds(), e.target.value);
        });
    }
    const pClear = document.getElementById('travel-participants-clear');
    if (pClear) {
        pClear.addEventListener('click', () => {
            renderTravelParticipants(travelEmpsCache, [], '');
        });
    }

    // --- Filter bar ---
    document.querySelectorAll('.travel-filter-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.travel-filter-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            travelPage = 1;
            travelCurrentFilter = btn.dataset.status;
            loadTravelAllowances();
        });
    });

    // --- Travel form submit ---
    const travelForm = document.getElementById('travel-form');
    if (travelForm) {
        travelForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const id = document.getElementById('travel-id').value;
            const employeeIds = getTravelSelectedEmployeeIds();
            const payload = {
                rateId: document.getElementById('travel-rate').value,
                destination: document.getElementById('travel-destination').value,
                departureDate: document.getElementById('travel-departure').value + 'T00:00:00Z',
                returnDate: document.getElementById('travel-return').value + 'T00:00:00Z',
                reason: document.getElementById('travel-reason').value,
            };
            if (employeeIds.length > 1) {
                payload.employeeIds = employeeIds;
                payload.groupName = document.getElementById('travel-group-name').value.trim();
            } else {
                payload.employeeId = employeeIds[0] || '';
            }

            const btn = document.getElementById('travel-submit-btn');
            const orig = btn.innerText;
            btn.disabled = true;
            btn.innerText = 'Guardando...';

            try {
                const url = id ? `/api/travel-allowances/${id}` : '/api/travel-allowances';
                const method = id ? 'PUT' : 'POST';
                const resp = await fetch(url, {
                    method,
                    headers: getAuthHeaders(),
                    body: JSON.stringify(payload)
                });
                if (resp.ok) {
                    showToast(id ? 'Solicitud actualizada' : employeeIds.length > 1 ? 'Solicitud grupal creada correctamente' : 'Solicitud creada correctamente');
                    document.getElementById('travel-modal').classList.remove('active');
                    loadTravelAllowances();
                } else {
                    const err = await resp.json();
                    showToast(err.error || 'Error al guardar solicitud', 'error');
                }
            } catch (err) {
                showToast('Error de conexión', 'error');
            } finally {
                btn.disabled = false;
                btn.innerText = orig;
            }
        });
    }

    // --- Calc preview: update on change ---
    ['travel-employee', 'travel-rate', 'travel-departure', 'travel-return', 'travel-employees-group'].forEach(id => {
        const el = document.getElementById(id);
        if (el) el.addEventListener('change', updateTravelCalcPreview);
    });

    // --- Rate type label ---
    const rateTypeSelect = document.getElementById('rate-type');
    if (rateTypeSelect) {
        rateTypeSelect.addEventListener('change', () => {
            const label = document.getElementById('rate-value-label');
            if (label) label.textContent = rateTypeSelect.value === 'percentage'
                ? 'Porcentaje del salario diario (%)'
                : 'Monto fijo por día (RD$)';
        });
    }

    // --- Rate form ---
    const rateForm = document.getElementById('rate-form');
    if (rateForm) {
        rateForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const id = document.getElementById('rate-id').value;
            const payload = {
                name: document.getElementById('rate-name').value,
                type: document.getElementById('rate-type').value,
                value: parseFloat(document.getElementById('rate-value').value),
                active: true
            };
            try {
                const url = id ? `/api/travel-rates/${id}` : '/api/travel-rates';
                const method = id ? 'PUT' : 'POST';
                const resp = await fetch(url, {
                    method,
                    headers: getAuthHeaders(),
                    body: JSON.stringify(payload)
                });
                if (resp.ok) {
                    showToast(id ? 'Tarifa actualizada' : 'Tarifa creada');
                    resetRateForm();
                    loadRates();
                } else {
                    const err = await resp.json();
                    showToast(err.error || 'Error al guardar tarifa', 'error');
                }
            } catch (err) {
                showToast('Error de conexión', 'error');
            }
        });
    }

    const btnRateCancel = document.getElementById('btn-rate-cancel');
    if (btnRateCancel) btnRateCancel.addEventListener('click', resetRateForm);

    // --- Decision modal confirm ---
    const btnConfirmDecision = document.getElementById('btn-confirm-decision');
    if (btnConfirmDecision) {
        btnConfirmDecision.addEventListener('click', async () => {
            if (!travelDecisionId || !travelDecisionType) return;
            const notes = document.getElementById('decision-notes').value;
            const endpoint = travelDecisionType === 'approve' ? 'approve' : 'reject';
            try {
                const resp = await fetch(`/api/travel-allowances/${travelDecisionId}/${endpoint}`, {
                    method: 'POST',
                    headers: getAuthHeaders(),
                    body: JSON.stringify({ notes })
                });
                if (resp.ok) {
                    const label = travelDecisionType === 'approve' ? 'aprobada' : 'rechazada';
                    showToast(`Solicitud ${label} correctamente`);
                    document.getElementById('travel-decision-modal').classList.remove('active');
                    loadTravelAllowances();
                } else {
                    const err = await resp.json();
                    showToast(err.error || 'Error al procesar decisión', 'error');
                }
            } catch (err) {
                showToast('Error de conexión', 'error');
            }
        });
    }

    setTravelRequestTypeUI();
}

async function loadTravelAllowances() {
    const tbody = document.getElementById('travel-list');
    if (!tbody) return;
    try {
        const resp = await fetch('/api/travel-allowances', { headers: getAuthHeaders() });
        if (!resp.ok) {
            const err = await safeReadJSON(resp);
            tbody.innerHTML = `<tr><td colspan="9" style="text-align:center;padding:2rem;color:var(--danger);">${escapeHTML(err.error || 'No fue posible cargar la tabla de viáticos.')}</td></tr>`;
            return;
        }
        const data = await resp.json();
        window.allTravelData = data || [];
        
        const activeFilter = document.querySelector('.travel-filter-btn.active')?.dataset.status || 'all';
        renderTravelTable(window.allTravelData, activeFilter);
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="9" style="text-align:center;padding:2rem;color:var(--danger);">Error de conexión</td></tr>`;
    }
}

function renderTravelTable(data, filter = 'all') {
    const tbody = document.getElementById('travel-list');
    if (!tbody) return;

    let filtered = data;
    if (filter !== 'all') {
        filtered = data.filter(t => t.status === filter);
    }

    if (!filtered || filtered.length === 0) {
        tbody.innerHTML = `<tr><td colspan="9" style="text-align:center;padding:2rem;color:var(--text-muted);">No hay solicitudes de viáticos ${filter !== 'all' ? 'con estado ' + filter : 'registradas'}.</td></tr>`;
        document.getElementById('travel-row-count').textContent = 'Mostrando 0 de 0 registros';
        document.getElementById('travel-pagination').innerHTML = '';
        return;
    }

    const totalPages = Math.ceil(filtered.length / PAGINATION_SIZE);
    if (travelPage > totalPages) travelPage = totalPages || 1;
    const start = (travelPage - 1) * PAGINATION_SIZE;
    const end = start + PAGINATION_SIZE;
    const pageItems = filtered.slice(start, end);

    const isManager = canManageOrganization();

    tbody.innerHTML = pageItems.map(t => {
        const depDate = new Date(t.departureDate).toLocaleDateString('es-ES', { day: '2-digit', month: 'short', year: 'numeric' });
        const retDate = new Date(t.returnDate).toLocaleDateString('es-ES', { day: '2-digit', month: 'short', year: 'numeric' });
        const statusClass = `badge-${t.status?.toLowerCase() || 'secondary'}`;
        const employeeName = t.employeeName || (t.isGroup ? `Grupo: ${t.groupName}` : '---');

        return `
        <tr>
            <td>${escapeHTML(employeeName)}</td>
            <td>${escapeHTML(t.destination || '---')}</td>
            <td>${depDate}</td>
            <td>${retDate}</td>
            <td>${t.totalDays || 0}</td>
            <td>${t.dailyRate?.toFixed(2) || '0.00'}</td>
            <td><strong>${t.totalAmount?.toFixed(2) || '0.00'}</strong></td>
            <td><span class="badge ${statusClass}">${escapeHTML(t.status || 'Pending')}</span></td>
            <td>
                <div class="travel-actions">
                    ${t.status === 'Pending' && isManager ? `
                        <button class="btn-action btn-action--icon btn-action--view" onclick="decideTravel('${t.id}', 'Approved')" title="Aprobar"><svg style="width:14px;height:14px;"><use href="#icon-check"></use></svg></button>
                        <button class="btn-action btn-action--icon btn-action--danger" onclick="decideTravel('${t.id}', 'Rejected')" title="Rechazar"><svg style="width:14px;height:14px;"><use href="#icon-x"></use></svg></button>
                    ` : ''}
                    <button class="btn-action btn-action--icon btn-action--primary" onclick="viewTravelDetails('${t.id}')" title="Ver Detalles"><svg style="width:14px;height:14px;"><use href="#icon-dashboard"></use></svg></button>
                </div>
            </td>
        </tr>
        `;
    }).join('');

    renderCommonPagination('travel-pagination', filtered.length, travelPage, 'changeTravelPage', 'travel-row-count');
}

window.changeTravelPage = (p) => {
    travelPage = p;
    const filter = document.querySelector('.travel-filter-btn.active')?.dataset.status || 'all';
    renderTravelTable(window.allTravelData, filter);
    document.querySelector('#travel-allowances .users-table-container').scrollTop = 0;
};

async function openTravelModal(id = null) {
    const modal = document.getElementById('travel-modal');
    const form = document.getElementById('travel-form');
    if (!modal || !form) return;

    form.reset();
    document.getElementById('travel-id').value = '';
    document.getElementById('travel-calc-preview').style.display = 'none';
    document.getElementById('travel-request-type').value = 'single';
    document.getElementById('travel-request-type').disabled = false;
    document.getElementById('travel-group-name').value = '';
    
    // Show modal immediately
    modal.classList.add('active');
    setTravelRequestTypeUI();

    // Background loading
    try {
        await Promise.all([loadTravelEmployeeSelect(), loadTravelRateSelect()]);
    } catch(e) {
        console.error('Error loading travel requirements', e);
    }

    if (id) {
        try {
            const resp = await fetch(`/api/travel-allowances/${id}`, { headers: getAuthHeaders() });
            if (!resp.ok) { showToast('No se pudo cargar la solicitud', 'error'); return; }
            const ta = await resp.json();
            document.getElementById('travel-id').value = ta.id;
            document.getElementById('travel-request-type').value = ta.groupSize > 1 ? 'group' : 'single';
            document.getElementById('travel-employee').value = ta.employeeId;
            document.getElementById('travel-group-name').value = ta.groupName || '';
            document.getElementById('travel-rate').value = ta.rateId;
            document.getElementById('travel-destination').value = ta.destination;
            document.getElementById('travel-departure').value = ta.departureDate.split('T')[0];
            document.getElementById('travel-return').value = ta.returnDate.split('T')[0];
            document.getElementById('travel-reason').value = ta.reason || '';
            
            if (ta.groupSize > 1 || (ta.employeeIds && ta.employeeIds.length > 0)) {
                document.getElementById('travel-request-type').value = 'group';
                const ids = ta.employeeIds || [ta.employeeId];
                renderTravelParticipants(travelEmpsCache, ids);
                document.getElementById('travel-request-type').disabled = true;
            } else {
                document.getElementById('travel-request-type').value = 'single';
                document.getElementById('travel-employee').value = ta.employeeId;
            }
            
            setTravelRequestTypeUI();
            document.getElementById('travel-modal-title').innerText = 'Editar Solicitud de Viático';
            updateTravelCalcPreview();
        } catch (err) {
            showToast('Error de conexión', 'error');
        }
    } else {
        document.getElementById('travel-modal-title').innerText = 'Nueva Solicitud de Viático';
    }
}

async function loadTravelEmployeeSelect() {
    try {
        const resp = await fetch('/api/employees', { headers: getAuthHeaders() });
        if (!resp.ok) return;
        const emps = await resp.json();
        travelEmpsCache = emps || [];
        const singleSelect = document.getElementById('travel-employee');
        if (!singleSelect) return;

        const currentSingle = singleSelect.value;
        const optionsHTML = travelEmpsCache.map(e => `<option value="${escapeHTML(e.id || '')}">${escapeHTML(`${e.firstName || ''} ${e.lastName || ''}`.trim())}</option>`).join('');

        singleSelect.innerHTML = '<option value="">Seleccionar empleado...</option>' + optionsHTML;
        singleSelect.value = currentSingle;

        // Initialize group lists if they are empty
        const selectedIds = getTravelSelectedEmployeeIds() || [];
        renderTravelParticipants(travelEmpsCache, selectedIds);
    } catch (err) { console.error('loadTravelEmployeeSelect', err); }
}

async function loadTravelRateSelect() {
    try {
        const resp = await fetch('/api/travel-rates', { headers: getAuthHeaders() });
        if (!resp.ok) return;
        const rates = await resp.json();
        travelRatesCache = (rates || []).filter(r => r.active);
        const sel = document.getElementById('travel-rate');
        if (!sel) return;
        const cur = sel.value;
        sel.innerHTML = '<option value="">Seleccionar tarifa...</option>' +
            travelRatesCache.map(r => {
                const typeLabel = r.type === 'percentage' ? `${r.value}%` : `RD$ ${r.value}/día`;
                return `<option value="${escapeHTML(r.id || '')}">${escapeHTML(`${r.name || ''} (${typeLabel})`)}</option>`;
            }).join('');
        sel.value = cur;
    } catch (err) { console.error('loadTravelRateSelect', err); }
}

function updateTravelCalcPreview() {
    const employeeIds = getTravelSelectedEmployeeIds() || [];
    const rateId = document.getElementById('travel-rate')?.value;
    const depVal = document.getElementById('travel-departure')?.value;
    const retVal = document.getElementById('travel-return')?.value;
    const preview = document.getElementById('travel-calc-preview');

    if (!employeeIds.length || !rateId || !depVal || !retVal) {
        if (preview) preview.style.display = 'none';
        return;
    }

    const dep = new Date(depVal);
    const ret = new Date(retVal);
    if (ret < dep) { if (preview) preview.style.display = 'none'; return; }

    const days = Math.round((ret - dep) / 86400000) + 1;
    const selectedEmployees = travelEmpsCache.filter(e => employeeIds.includes(e.id));
    const rate = travelRatesCache.find(r => r.id === rateId);

    if (!selectedEmployees.length || !rate) { if (preview) preview.style.display = 'none'; return; }

    const totalBaseSalary = selectedEmployees.reduce((sum, emp) => sum + (emp.baseSalary || 0), 0);
    const averageDailySalary = (totalBaseSalary / selectedEmployees.length) / 23.83;
    let amount;
    if (rate.type === 'percentage') {
        amount = selectedEmployees.reduce((sum, emp) => sum + (((emp.baseSalary || 0) / 23.83) * (rate.value / 100) * days), 0);
    } else {
        amount = rate.value * days * selectedEmployees.length;
    }
    amount = Math.round(amount * 100) / 100;

    document.getElementById('travel-calc-days').innerText = `${days} día${days !== 1 ? 's' : ''}`;
    document.getElementById('travel-calc-daily').innerText = rate.type === 'percentage'
        ? `Prom. RD$ ${averageDailySalary.toLocaleString('es-DO', { minimumFractionDigits: 2 })}`
        : `RD$ ${rate.value.toLocaleString('es-DO', { minimumFractionDigits: 2 })}/día x ${selectedEmployees.length}`;
    document.getElementById('travel-calc-amount').innerText = `RD$ ${amount.toLocaleString('es-DO', { minimumFractionDigits: 2 })}`;
    if (preview) preview.style.display = 'grid';
}

async function deleteTravelAllowance(id) {
    if (!confirm('¿Seguro que deseas eliminar esta solicitud? Esta acción no se puede deshacer.')) return;
    try {
        const resp = await fetch(`/api/travel-allowances/${id}`, {
            method: 'DELETE',
            headers: getAuthHeaders()
        });
        if (resp.ok || resp.status === 204) {
            showToast('Solicitud eliminada');
            loadTravelAllowances();
        } else {
            const err = await resp.json();
            showToast(err.error || 'Error al eliminar', 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
}

function openTravelDecision(id, type, label) {
    travelDecisionId = id;
    travelDecisionType = type;
    const isApprove = type === 'approve';
    document.getElementById('decision-modal-title').innerText = isApprove ? 'Aprobar Solicitud' : 'Rechazar Solicitud';
    document.getElementById('decision-modal-desc').innerText =
        `${isApprove ? 'Aprobarás' : 'Rechazarás'} la solicitud de viático de: ${label}. Puedes agregar notas opcionales.`;
    document.getElementById('decision-notes').value = '';
    const confirmBtn = document.getElementById('btn-confirm-decision');
    if (confirmBtn) {
        confirmBtn.style.background = isApprove ? '#10b981' : '#ef4444';
        confirmBtn.innerText = isApprove ? 'Aprobar' : 'Rechazar';
    }
    document.getElementById('travel-decision-modal').classList.add('active');
}

// Rates modal
async function openRatesModal() {
    if (!canManageTravelRates()) {
        showToast('Solo los administradores pueden gestionar tarifas', 'error');
        return;
    }
    resetRateForm();
    await loadRates();
    document.getElementById('rates-modal').classList.add('active');
}

async function loadRates() {
    const tbody = document.getElementById('rates-list');
    if (!tbody) return;
    try {
        const resp = await fetch('/api/travel-rates', { headers: getAuthHeaders() });
        if (!resp.ok) {
            const err = await safeReadJSON(resp);
            tbody.innerHTML = `<tr><td colspan="5" style="text-align:center;padding:1.5rem;color:var(--danger);">${escapeHTML(err.error || 'No fue posible cargar las tarifas.')}</td></tr>`;
            return;
        }
        const rates = await resp.json();
        travelRatesCache = rates || [];
        if (!travelRatesCache.length) {
            tbody.innerHTML = `<tr><td colspan="5" style="text-align:center;padding:1.5rem;color:var(--text-muted);">Sin tarifas configuradas.</td></tr>`;
            return;
        }
        tbody.innerHTML = travelRatesCache.map(r => {
            const typeBadge = `<span class="rate-type-badge rate-type-${escapeHTML(r.type || '')}">${r.type === 'percentage' ? 'Porcentual' : 'Fijo'}</span>`;
            const valLabel = r.type === 'percentage' ? `${r.value}%` : `RD$ ${r.value.toLocaleString('es-DO', { minimumFractionDigits: 2 })} / día`;
            const activeLabel = r.active
                ? `<span style="color:var(--success);font-size:0.8rem;">● Activa</span>`
                : `<span style="color:var(--text-muted);font-size:0.8rem;">● Inactiva</span>`;
            return `<tr>
                <td style="font-weight:500;">${escapeHTML(r.name || '')}</td>
                <td>${typeBadge}</td>
                <td>${escapeHTML(valLabel)}</td>
                <td>${activeLabel}</td>
                <td class="travel-actions-cell">
                    <div class="travel-actions">
                        <button class="btn-action btn-action--primary" onclick="editRate(decodeInlineArg('${encodeInlineArg(r.id || '')}'))" title="Editar"><svg style="width:16px;height:16px;"><use href="#icon-edit"></use></svg></button>
                        <button class="btn-action btn-action--danger" onclick="deleteRate(decodeInlineArg('${encodeInlineArg(r.id || '')}'))" title="Eliminar"><svg style="width:16px;height:16px;"><use href="#icon-trash"></use></svg></button>
                    </div>
                </td>
            </tr>`;
        }).join('');
    } catch (err) {
        console.error('loadRates', err);
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;padding:1.5rem;color:var(--danger);">Error de conexión al cargar las tarifas.</td></tr>';
    }
}

window.editRate = function(id) {
    const rate = travelRatesCache.find(r => r.id === id);
    if (!rate) return;
    document.getElementById('rate-id').value = rate.id;
    document.getElementById('rate-name').value = rate.name;
    document.getElementById('rate-type').value = rate.type;
    document.getElementById('rate-value').value = rate.value;
    const label = document.getElementById('rate-value-label');
    if (label) label.textContent = rate.type === 'percentage' ? 'Porcentaje del salario diario (%)' : 'Monto fijo por día (RD$)';
    const submitBtn = document.getElementById('rate-submit-btn');
    if (submitBtn) submitBtn.innerText = 'Actualizar Tarifa';
    const cancelBtn = document.getElementById('btn-rate-cancel');
    if (cancelBtn) cancelBtn.style.display = 'inline-flex';
};

window.deleteRate = async function(id) {
    if (!confirm('¿Eliminar esta tarifa? Las solicitudes existentes no se verán afectadas.')) return;
    try {
        const resp = await fetch(`/api/travel-rates/${id}`, {
            method: 'DELETE',
            headers: getAuthHeaders()
        });
        if (resp.ok || resp.status === 204) {
            showToast('Tarifa eliminada');
            loadRates();
        } else {
            const err = await resp.json();
            showToast(err.error || 'Error al eliminar tarifa', 'error');
        }
    } catch (err) {
        showToast('Error de conexión', 'error');
    }
};

window.openTravelModal = openTravelModal;
window.openTravelDecision = openTravelDecision;
window.deleteTravelAllowance = deleteTravelAllowance;
window.downloadTravelAllowancePDF = function(id) {
    const url = `/api/travel-allowances/${id}/pdf`;
    window.open(url, '_blank');
};

function resetRateForm() {
    const form = document.getElementById('rate-form');
    if (form) form.reset();
    document.getElementById('rate-id').value = '';
    const submitBtn = document.getElementById('rate-submit-btn');
    if (submitBtn) submitBtn.innerText = '+ Agregar Tarifa';
    const cancelBtn = document.getElementById('btn-rate-cancel');
    if (cancelBtn) cancelBtn.style.display = 'none';
    const label = document.getElementById('rate-value-label');
    if (label) label.textContent = 'Monto por día (RD$)';
}

function travelStatusBadge(status) {
    const map = {
        'Pending':  ['travel-badge-pending',  'Pendiente'],
        'Approved': ['travel-badge-approved', 'Aprobado'],
        'Rejected': ['travel-badge-rejected', 'Rechazado'],
    };
    const [cls, label] = map[status] || ['travel-badge-pending', status];
    return `<span class="travel-badge ${cls}">${label}</span>`;
}

function formatTravelStatus(s) {
    return s === 'Pending' ? 'Pendiente' : s === 'Approved' ? 'Aprobado' : s === 'Rejected' ? 'Rechazado' : s;
}

// ==================== REPORTE DE ASISTENCIA EN VIVO ====================

function initAttendanceReport() {
    // ── Elements ─────────────────────────────────────────────────────────────
    const applyBtn      = document.getElementById('ar-apply');
    const excelBtn      = document.getElementById('ar-export-excel');
    const pdfBtn        = document.getElementById('ar-export-pdf');
    const syncBtn       = document.getElementById('ar-sync-btn');
    const fromInput     = document.getElementById('ar-from');
    const toInput       = document.getElementById('ar-to');
    const deptSel       = document.getElementById('ar-dept');
    const statusSel     = document.getElementById('ar-status');
    const searchInp     = document.getElementById('ar-search');
    const loading       = document.getElementById('ar-loading');
    const emptyEl       = document.getElementById('ar-empty');
    const table         = document.getElementById('ar-table');
    const tbody         = document.getElementById('ar-tbody');
    const footer        = document.getElementById('ar-footer');
    const rowCount      = document.getElementById('ar-row-count');
    const paginationEl  = document.getElementById('ar-pagination');
    const refreshSel    = document.getElementById('ar-refresh-interval');
    const countdownEl   = document.getElementById('ar-refresh-countdown');

    if (!applyBtn) return;

    // ── State ─────────────────────────────────────────────────────────────────
    let allRows        = [];   // All rows from last API call
    let currentPage    = 1;
    const pageSize     = 20;
    let sortCol        = 'date';
    let sortAsc        = true;
    let refreshTimer   = null;
    let countdownTimer = null;
    let nextRefreshIn  = 0;

    // ── Defaults (Local Time) ─────────────────────────────────────────────────
    const formatDate = (d) => d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0');
    const setAttendanceTodayRange = () => {
        const today = formatDate(new Date());
        fromInput.value = today;
        toInput.value = today;
    };

    setAttendanceTodayRange();

    window.loadAttendance = () => {
        setAttendanceTodayRange();
        loadDepts();
        fetchData();
        setupAutoRefresh();
    };

    // ── Department loader ─────────────────────────────────────────────────────
    async function loadDepts() {
        while (deptSel.options.length > 1) deptSel.remove(1);
        try {
            const resp = await fetch('/api/departments', { headers: getAuthHeaders() });
            if (!resp.ok) return;
            const depts = await resp.json();
            deptSel.innerHTML = '<option value="">Todos</option>' + 
                depts.map(d => `<option value="${escapeHTML(d.id || '')}">${escapeHTML(d.name || '')}</option>`).join('');
        } catch (e) { /* silent */ }
    }

    // ── Data fetch ────────────────────────────────────────────────────────────
    async function fetchData(triggerSync = false) {
        const from = fromInput.value;
        const to   = toInput.value;
        if (!from || !to) { showToast('Selecciona el rango de fechas', 'error'); return; }
        if (from > to)    { showToast('Fecha inicio mayor a fecha fin', 'error'); return; }

        if (triggerSync) {
            // Trigger a quick read from devices before fetching report data
            try {
                const syncResp = await fetch(`/api/devices/read-events?from=${from}&to=${to}`, { method: 'POST', headers: getAuthHeaders() });
                const syncData = await syncResp.json();
                if (syncResp.ok && syncData.eventsRead > 0) {
                    showToast(`Sincronización finalizada: ${syncData.eventsRead} eventos nuevos procesados`, 'success');
                }
            } catch (e) { console.warn('Device sync failed during refresh', e); }
        }

        loading.style.display = 'flex';
        emptyEl.style.display = 'none';
        table.style.display   = 'none';
        footer.style.display  = 'none';

        let url = `/api/reports/attendance/data?from=${from}&to=${to}`;
        const dept = deptSel.value;
        if (dept) url += `&department=${encodeURIComponent(dept)}`;

        try {
            const resp = await fetch(url, { headers: getAuthHeaders() });
            if (!resp.ok) throw new Error(await resp.text());
            const data = await resp.json();

            allRows = data.rows || [];
            updateKPIs(data.summary || {});
            
            currentPage = 1; // Reset to page 1 on new fetch
            loading.style.display = 'none';
            renderTable();
        } catch (err) {
            console.error('Fetch error:', err);
            showToast('Error al cargar datos: ' + err.message, 'error');
            loading.style.display = 'none';
        }
    }

    // ── Auto Refresh Logic ──────────────────────────────────────────────────
    function setupAutoRefresh() {
        if (refreshTimer) clearInterval(refreshTimer);
        if (countdownTimer) clearInterval(countdownTimer);
        
        const interval = parseInt(refreshSel.value);
        if (interval <= 0) {
            countdownEl.textContent = '';
            return;
        }

        nextRefreshIn = interval;
        updateCountdownDisplay();

        countdownTimer = setInterval(() => {
            nextRefreshIn--;
            if (nextRefreshIn <= 0) {
                nextRefreshIn = interval;
                fetchData(true); // Trigger sync on auto-refresh
            }
            updateCountdownDisplay();
        }, 1000);
    }

    function updateCountdownDisplay() {
        const m = Math.floor(nextRefreshIn / 60);
        const s = nextRefreshIn % 60;
        countdownEl.textContent = `${m}:${s < 10 ? '0' : ''}${s}`;
    }

    refreshSel.addEventListener('change', setupAutoRefresh);

    applyBtn.addEventListener('click', () => fetchData());
    
    if (syncBtn) {
        syncBtn.addEventListener('click', async () => {
            syncBtn.disabled = true;
            syncBtn.innerHTML = `<svg viewBox="0 0 24 24" aria-hidden="true" style="width:15px;height:15px" class="spin"><use href="#icon-refresh"></use></svg> Sincronizando...`;
            showToast('Sincronizando con los dispositivos biometrícos...', 'info');
            await fetchData(true);
            syncBtn.disabled = false;
            syncBtn.innerHTML = `<svg viewBox="0 0 24 24" aria-hidden="true" style="width:15px;height:15px"><use href="#icon-refresh"></use></svg> Sincronizar`;
        });
    }
    fromInput.addEventListener('change', () => fetchData());
    toInput.addEventListener('change',   () => fetchData());
    deptSel.addEventListener('change',   () => fetchData());

    // ── KPIs ──────────────────────────────────────────────────────────────────
    function updateKPIs(s) {
        const set = (id, val) => { const el = document.getElementById(id); if (el) el.textContent = val; };
        set('ar-kpi-total',    s.total      ?? 0);
        set('ar-kpi-present',  s.present    ?? 0);
        set('ar-kpi-late',     s.late       ?? 0);
        set('ar-kpi-absent',   s.absent     ?? 0);
        set('ar-kpi-hours',    (s.totalHours ?? 0).toFixed(1));
        set('ar-kpi-overtime', (s.totalOvertime ?? 0).toFixed(1));
    }

    // ── Client-side filter + sort + pagination ────────────────────────────────
    function getFiltered() {
        const q      = (searchInp.value || '').toLowerCase();
        const status = statusSel.value;

        return allRows.filter(row => {
            if (status && row.status !== status) return false;
            if (q) {
                const hay = (row.employeeName + ' ' + row.employeeNo).toLowerCase();
                if (!hay.includes(q)) return false;
            }
            return true;
        });
    }

    searchInp.addEventListener('input', () => { currentPage = 1; renderTable(); });
    statusSel.addEventListener('change', () => { currentPage = 1; renderTable(); });

    function renderTable() {
        loading.style.display = 'none';
        const filtered = getFiltered();

        if (filtered.length === 0) {
            emptyEl.style.display = 'flex';
            table.style.display   = 'none';
            footer.style.display  = 'none';
            return;
        }

        // Sort
        const mult = sortAsc ? 1 : -1;
        filtered.sort((a, b) => {
            const va = a[sortCol] ?? '';
            const vb = b[sortCol] ?? '';
            if (typeof va === 'number') return (va - vb) * mult;
            return String(va).localeCompare(String(vb)) * mult;
        });

        // Paginate
        const totalPages = Math.ceil(filtered.length / pageSize);
        if (currentPage > totalPages) currentPage = totalPages || 1;
        
        const start = (currentPage - 1) * pageSize;
        const end = start + pageSize;
        const pageRows = filtered.slice(start, end);

        emptyEl.style.display = 'none';
        table.style.display   = 'table';
        footer.style.display  = 'flex';

        tbody.innerHTML = pageRows.map(row => {
            const statusClass = {
                'Presente':   'ar-badge-present',
                'Tarde':      'ar-badge-late',
                'Falta':      'ar-badge-absent',
                'Incompleto': 'ar-badge-incomplete',
            }[row.status] || '';

            const lateDisplay = row.lateMinutes > 0 ? `${row.lateMinutes} min` : '—';
            const dateDisplay = row.date ? formatDateDisplay(row.date) : '—';

            return `<tr>
                <td class="ar-td ar-td-mono">${escapeHTML(row.employeeNo || '')}</td>
                <td class="ar-td ar-td-name">${escapeHTML(row.employeeName || '')}</td>
                <td class="ar-td">${row.department || '—'}</td>
                <td class="ar-td ar-td-mono">${dateDisplay}</td>
                <td class="ar-td ar-td-center">${row.checkIn  || '—'}</td>
                <td class="ar-td ar-td-center">${row.checkOut || '—'}</td>
                <td class="ar-td ar-td-right">${row.totalHours.toFixed(2)}</td>
                <td class="ar-td ar-td-right">${row.overtimeHrs > 0 ? row.overtimeHrs.toFixed(2) : '—'}</td>
                <td class="ar-td ar-td-center">${lateDisplay}</td>
                <td class="ar-td"><span class="ar-badge ${statusClass}">${row.status}</span></td>
                <td class="ar-td ar-td-center">
                    <div class="travel-actions" style="justify-content:center;">
                        <button class="btn-action btn-action--icon btn-action--danger" onclick='openNotifyModal(JSON.parse(decodeInlineArg("${encodeInlineArg(JSON.stringify(row))}")))' title="Reportar">
                            <svg viewBox="0 0 24 24" style="width:14px;height:14px;fill:currentColor"><use href="#icon-alert"></use></svg>
                        </button>
                    </div>
                </td>
            </tr>`;
        }).join('');

        rowCount.textContent = `Mostrando ${start + 1}-${Math.min(end, filtered.length)} de ${filtered.length} registro${filtered.length !== 1 ? 's' : ''}`;
        renderPagination(totalPages);
        updateSortIcons();
    }

    function renderPagination(totalPages) {
        if (totalPages <= 1) {
            paginationEl.innerHTML = '';
            return;
        }

        let html = `<button class="btn btn-secondary btn-sm" ${currentPage === 1 ? 'disabled' : ''} onclick="arChangePage(${currentPage - 1})">Anterior</button>`;
        
        const startPage = Math.max(1, currentPage - 2);
        const endPage = Math.min(totalPages, startPage + 4);
        
        for (let i = startPage; i <= endPage; i++) {
            html += `<button class="btn btn-sm ${i === currentPage ? 'btn-primary' : 'btn-secondary'}" onclick="arChangePage(${i})">${i}</button>`;
        }

        html += `<button class="btn btn-secondary btn-sm" ${currentPage === totalPages ? 'disabled' : ''} onclick="arChangePage(${currentPage + 1})">Siguiente</button>`;
        
        paginationEl.innerHTML = html;
    }

    window.arChangePage = (p) => {
        currentPage = p;
        renderTable();
        document.querySelector('.ar-table-wrap').scrollTop = 0;
    };

    function formatDateDisplay(dateStr) {
        const [y, m, d] = dateStr.split('-');
        return `${d}/${m}/${y}`;
    }

    // ── Column sorting ────────────────────────────────────────────────────────
    document.querySelectorAll('.ar-th.sortable').forEach(th => {
        th.addEventListener('click', () => {
            const col = th.dataset.col;
            if (sortCol === col) {
                sortAsc = !sortAsc;
            } else {
                sortCol = col;
                sortAsc = true;
            }
            renderTable();
        });
    });

    function updateSortIcons() {
        document.querySelectorAll('.ar-th.sortable').forEach(th => {
            const icon = th.querySelector('.sort-icon');
            if (!icon) return;
            if (th.dataset.col === sortCol) {
                icon.textContent = sortAsc ? ' ↑' : ' ↓';
                th.classList.add('ar-th-active');
            } else {
                icon.textContent = '';
                th.classList.remove('ar-th-active');
            }
        });
    }

    // ── Export (respects ALL current filters) ─────────────────────────────────
    function buildExportUrl(format) {
        const from   = fromInput.value;
        const to     = toInput.value;
        const dept   = deptSel.value;
        const status = statusSel.value;
        const search = searchInp.value;

        let url = `/api/reports/attendance?from=${from}&to=${to}&format=${format}`;
        if (dept)   url += `&department=${encodeURIComponent(dept)}`;
        if (status) url += `&status=${encodeURIComponent(status)}`;
        if (search) url += `&search=${encodeURIComponent(search)}`;
        
        return url;
    }

    excelBtn.addEventListener('click', () => {
        const from = fromInput.value, to = toInput.value;
        downloadReport(buildExportUrl('excel'), `reporte_asistencia_${from}_${to}.xlsx`);
    });

    pdfBtn.addEventListener('click', () => {
        const from = fromInput.value, to = toInput.value;
        downloadReport(buildExportUrl('pdf'), `reporte_asistencia_${from}_${to}.pdf`);
    });
}

window.showReportPeriodModal = async () => {
    const modal = document.getElementById('attendance-period-modal');
    if (modal) {
        // Preset dates if empty
        const from = document.getElementById('period-from');
        const to = document.getElementById('period-to');
        if (from && !from.value) {
            const now = new Date();
            const first = new Date(now.getFullYear(), now.getMonth(), 1);
            from.value = first.toISOString().split('T')[0];
            to.value = now.toISOString().split('T')[0];
        }

        // Load departments dynamically to avoid auth state issues
        const deptSel = document.getElementById('period-dept');
        if (deptSel) {
            try {
                const resp = await fetch('/api/departments', { headers: getAuthHeaders() });
                const depts = await resp.json();
                deptSel.innerHTML = '<option value="">Todos</option>' + 
                    depts.map(d => `<option value="${escapeHTML(d.id || '')}">${escapeHTML(d.name || '')}</option>`).join('');
            } catch(e) {}
        }

        modal.classList.add('active');
    } else {
        const attendanceBtn = document.querySelector('.sidebar nav li[data-page="attendance"]');
        if (attendanceBtn) attendanceBtn.click();
    }
};

// Hook modal buttons for reporting
document.addEventListener('DOMContentLoaded', () => {
    const pdfBtn = document.getElementById('btn-period-pdf');
    const excelBtn = document.getElementById('btn-period-excel');
    const fromInp = document.getElementById('period-from');
    const toInp = document.getElementById('period-to');
    const deptSel = document.getElementById('period-dept');

    // Load depts into modal select
    const loadDepts = async () => {
        if (!deptSel) return;
        try {
            const resp = await fetch('/api/departments', { headers: getAuthHeaders() });
            const depts = await resp.json();
            deptSel.innerHTML = '<option value="">Todos</option>' + 
                depts.map(d => `<option value="${escapeHTML(d.id || '')}">${escapeHTML(d.name || '')}</option>`).join('');
        } catch(e) {}
    };
    loadDepts();

    const download = (format) => {
        const from = fromInp.value;
        const to = toInp.value;
        const dept = deptSel.value;
        if (!from || !to) { showToast('Selecciona el rango de fechas', 'error'); return; }
        
        let url = `/api/reports/attendance?from=${from}&to=${to}&format=${format}`;
        if (dept) url += `&department=${encodeURIComponent(dept)}`;
        
        showToast(`Generando reporte ${format.toUpperCase()}...`);
        window.open(url, '_blank');
        document.getElementById('attendance-period-modal').classList.remove('active');
    };

    pdfBtn?.addEventListener('click', () => download('pdf'));
    excelBtn?.addEventListener('click', () => download('excel'));
    
    document.querySelector('.close-attendance-period')?.addEventListener('click', () => {
        document.getElementById('attendance-period-modal').classList.remove('active');
    });
    document.getElementById('close-attendance-period-modal')?.addEventListener('click', () => {
        document.getElementById('attendance-period-modal').classList.remove('active');
    });
});

// ==================== NOTIFICATIONS ====================

let currentNotifyData = null;

window.openNotifyModal = async (rowData) => {
    currentNotifyData = rowData;
    document.getElementById('notify-dept-name').innerText = rowData.department || 'Administración';
    document.getElementById('notify-notes').value = '';
    
    // Fetch department to get manager contact (Not implemented completely in UI fetch, assuming backend gets it)
    let managerPhone = '';
    let managerEmail = '';
    try {
        const dResp = await fetch(`/api/departments`, { headers: getAuthHeaders() });
        const depts = await dResp.json();
        const dept = depts.find(d => d.name === rowData.department);
        if (dept && dept.managerId) {
            const eResp = await fetch(`/api/employees/${dept.managerId}`, { headers: getAuthHeaders() });
            const mgr = await eResp.json();
            managerPhone = mgr.phone || '';
            managerEmail = mgr.email || '';
        }
    } catch(e) {}

    currentNotifyData.managerPhone = managerPhone;
    currentNotifyData.managerEmail = managerEmail;
    
    document.getElementById('notify-modal').classList.add('active');
};

async function buildNotificationLinks() {
    if (!currentNotifyData) return null;
    currentNotifyData.notes = document.getElementById('notify-notes').value;
    try {
        const resp = await fetch('/api/notify/employee', {
            method: 'POST',
            headers: getAuthHeaders(),
            body: JSON.stringify(currentNotifyData)
        });
        if (resp.ok) return await resp.json();
        showToast('Error al preparar notificación', 'error');
    } catch(err) { showToast('Error de conexión', 'error'); }
    return null;
}

document.getElementById('btn-notify-wa')?.addEventListener('click', async () => {
    const data = await buildNotificationLinks();
    if (data && data.whatsappUrl) {
        window.open(data.whatsappUrl, '_blank');
        document.getElementById('notify-modal').classList.remove('active');
    } else if (data) {
        showToast('El encargado no tiene teléfono registrado', 'warning');
    }
});

document.getElementById('btn-notify-mail')?.addEventListener('click', async () => {
    const data = await buildNotificationLinks();
    if (data && data.mailtoUrl) {
        window.open(data.mailtoUrl, '_self');
        document.getElementById('notify-modal').classList.remove('active');
    } else if (data) {
        showToast('El encargado no tiene email registrado', 'warning');
    }
});

// ==================== LEAVES (PERMISOS Y AUSENCIAS) ====================

function initLeavesUI() {
    const modal = document.getElementById('leave-modal');
    const form = document.getElementById('leave-form');
    if (!modal || !form) return;

    modal.querySelectorAll('.close-modal').forEach(btn => {
        btn.addEventListener('click', () => modal.classList.remove('active'));
    });

    document.querySelectorAll('.leave-filter-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.leave-filter-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            leavesPage = 1; // Reset to page 1 on filter change
            loadLeaves();
        });
    });

    form?.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());
        data.days = parseInt(data.days) || 1;
        const id = data.id;
        delete data.id;
        
        try {
            const resp = await fetch(id ? `/api/leaves/${id}` : '/api/leaves', {
                method: id ? 'PUT' : 'POST',
                headers: getAuthHeaders(),
                body: JSON.stringify(data)
            });
            if (resp.ok) {
                showToast(id ? 'Permiso actualizado' : 'Permiso registrado');
                modal.classList.remove('active');
                loadLeaves();
            } else {
                const err = await resp.text();
                showToast(`Error: ${err}`, 'error');
            }
        } catch (err) {
            showToast('Error de red', 'error');
        }
    });
}

window.openNewLeaveModal = openNewLeaveModal;

async function openNewLeaveModal() {
    console.log("Opening new leave modal...");
    const modal = document.getElementById('leave-modal');
    const form = document.getElementById('leave-form');
    if (!modal || !form) {
        console.error("Leave modal or form not found");
        return;
    }

    document.getElementById('leave-modal-title').innerText = 'Nuevo Permiso';
    form.reset();
    if (document.getElementById('leave-id')) document.getElementById('leave-id').value = '';
    if (document.getElementById('leave-status')) document.getElementById('leave-status').value = 'Approved';

    // Open first so the UI responds even if the background fetch is slow.
    modal.classList.add('active');

    try {
        await loadLeaveSelects();
    } catch(e) {
        console.error('Error loading selects for new leave', e);
    }
}

async function loadLeaveSelects() {
    try {
        const [empResp, userResp] = await Promise.all([
            fetch('/api/employees', { headers: getAuthHeaders() }),
            fetch('/api/users', { headers: getAuthHeaders() })
        ]);
        
        const emps = empResp.ok ? await empResp.json() : [];
        const users = userResp.ok ? await userResp.json() : [];
        
        const empSel = document.getElementById('leave-employee');
        const authSel = document.getElementById('leave-authorized-by');
        
        if (empSel) {
            empSel.innerHTML = '<option value="">Seleccionar empleado...</option>' + 
                (Array.isArray(emps) ? emps.map(e => `<option value="${escapeHTML(e.id || '')}">${escapeHTML(`${e.firstName || ''} ${e.lastName || ''}`.trim())}</option>`).join('') : '');
        }
        if (authSel) {
            authSel.innerHTML = '<option value="">Seleccionar...</option>' + 
                (Array.isArray(users) ? users.map(u => `<option value="${escapeHTML(u.id || '')}">${escapeHTML(u.fullName || u.username || '')}</option>`).join('') : '');
        }
    } catch(e) {
        console.error('Error loading selects for leaves', e);
    }
}

async function loadLeaves() {
    try {
        const resp = await fetch('/api/leaves', { headers: getAuthHeaders() });
        const list = await resp.json();
        // store locally for filtering
        document.getElementById('leaves-table').leavesData = list;
        const activeFilter = document.querySelector('.leave-filter-btn.active')?.dataset.status || 'all';
        renderLeaves(list, activeFilter);
    } catch(e) { console.error('Failed to load leaves', e); }
}

function renderLeaves(leaves, filter = 'all') {
    const tbody = document.getElementById('leaves-list');
    if (!tbody) return;

    window.allLeavesData = leaves || []; // Globalize for callback

    let filtered = leaves || [];
    if (filter !== 'all') {
        filtered = filtered.filter(l => l.status === filter);
    }

    if (!filtered || filtered.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;padding:2rem;color:var(--text-muted);">No hay permisos registrados${filter !== 'all' ? ' con estado ' + filter : ''}.</td></tr>`;
        document.getElementById('leaves-row-count').textContent = 'Mostrando 0 de 0 registros';
        document.getElementById('leaves-pagination').innerHTML = '';
        return;
    }

    const totalPages = Math.ceil(filtered.length / PAGINATION_SIZE);
    if (leavesPage > totalPages) leavesPage = totalPages || 1;
    const start = (leavesPage - 1) * PAGINATION_SIZE;
    const end = start + PAGINATION_SIZE;
    const pageItems = filtered.slice(start, end);

    const isManager = canManageOrganization();
    const typeLabels = { 'Vacation': 'Vacaciones', 'Sick': 'Médico', 'Personal': 'Personal', 'Unpaid': 'Sin Goce', 'Other': 'Otro' };
    const statusBadges = {
        'Approved': '<span class="badge badge-success">Aprobado</span>',
        'Pending': '<span class="badge badge-warning">Pendiente</span>',
        'Rejected': '<span class="badge badge-danger">Rechazado</span>'
    };

    tbody.innerHTML = pageItems.map(l => {
        const canEditLeaves = canManageOrganization();
        return `
        <tr>
            <td>
                <strong>${escapeHTML(l.employeeName || '')}</strong>
                <div class="text-muted" style="font-size:0.75rem">${escapeHTML(l.department || '')}</div>
            </td>
            <td>${escapeHTML(typeLabels[l.type] || l.type || '')}</td>
            <td>
                ${l.startDate ? l.startDate.split('T')[0] : '—'}<br>
                <small class="text-muted">al ${l.endDate ? l.endDate.split('T')[0] : '—'}</small>
            </td>
            <td>${l.days}</td>
            <td>
                ${statusBadges[l.status]}<br>
                <small class="text-muted">${escapeHTML(l.authorizerName || '---')}</small>
            </td>
            <td class="travel-actions-cell">
                <div class="travel-actions">
                    ${canEditLeaves ? `<button class="btn-action btn-action--primary" onclick="editLeave(decodeInlineArg('${encodeInlineArg(l.id || '')}'))" title="Editar"><svg style="width:16px;height:16px;"><use href="#icon-edit"></use></svg></button>` : '<span class="text-muted" style="font-size:0.8rem">Solo lectura</span>'}
                    ${canEditLeaves ? `<button class="btn-action btn-action--danger" onclick="deleteLeave(decodeInlineArg('${encodeInlineArg(l.id || '')}'))" title="Eliminar"><svg style="width:16px;height:16px;"><use href="#icon-trash"></use></svg></button>` : ''}
                </div>
            </td>
        </tr>
        `;
    }).join('');

    renderCommonPagination('leaves-pagination', filtered.length, leavesPage, 'changeLeavesPage', 'leaves-row-count');
}

window.changeLeavesPage = (p) => {
    leavesPage = p;
    const filter = document.querySelector('.leave-filter-btn.active')?.dataset.status || 'all';
    renderLeaves(window.allLeavesData, filter);
    document.querySelector('#leaves .users-table-container').scrollTop = 0;
};

window.editLeave = async (id) => {
    const modal = document.getElementById('leave-modal');
    if (!modal) return;
    
    // Show modal immediately with a loading state or title update
    document.getElementById('leave-modal-title').innerText = 'Editar Permiso';
    modal.classList.add('active');
    
    try {
        const resp = await fetch(`/api/leaves/${id}`, { headers: getAuthHeaders() });
        if (!resp.ok) {
            showToast('No se pudo cargar el permiso', 'error');
            modal.classList.remove('active');
            return;
        }
        const l = await resp.json();
        
        await loadLeaveSelects();
        
        document.getElementById('leave-id').value = l.id || '';
        document.getElementById('leave-employee').value = l.employeeId || '';
        document.getElementById('leave-type').value = l.type || 'Vacation';
        document.getElementById('leave-status').value = l.status || 'Pending';
        document.getElementById('leave-start').value = l.startDate ? l.startDate.split('T')[0] : '';
        document.getElementById('leave-end').value = l.endDate ? l.endDate.split('T')[0] : '';
        document.getElementById('leave-days').value = l.days || 1;
        document.getElementById('leave-authorized-by').value = l.authorizedBy || '';
        document.getElementById('leave-reason').value = l.reason || '';
        document.getElementById('leave-notes').value = l.notes || '';
    } catch(e) {
        console.error('Error in editLeave:', e);
        showToast('Error de conexión', 'error');
        modal.classList.remove('active');
    }
};

window.deleteLeave = async (id) => {
    if (!confirm('¿Eliminar este permiso?')) return;
    try {
        const resp = await fetch(`/api/leaves/${id}`, { method: 'DELETE', headers: getAuthHeaders() });
        if (resp.ok) {
            showToast('Permiso eliminado');
            loadLeaves();
        }
    } catch(e) {}
};

window.openNewLeaveModal = openNewLeaveModal;

// ==================== HOLIDAYS ====================

async function loadHolidays() {
    try {
        const resp = await fetch('/api/holidays', { headers: getAuthHeaders() });
        const list = await resp.json();
        renderHolidays(list);
    } catch (err) {
        console.error('Failed to load holidays:', err);
        showToast('Error al cargar feriados', 'error');
    }
}

function renderHolidays(list) {
    const tbody = document.getElementById('holidays-list');
    const rowCount = document.getElementById('holidays-row-count');
    tbody.innerHTML = '';

    if (!list || list.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center; padding:2rem; color:var(--text-muted);">No hay feriados registrados</td></tr>';
        rowCount.innerText = 'Mostrando 0 de 0 registros';
        return;
    }

    list.forEach(h => {
        const tr = document.createElement('tr');
        const date = new Date(h.date).toLocaleDateString();
        tr.innerHTML = `
            <td><strong>${date}</strong></td>
            <td>${h.name}</td>
            <td><span class="text-muted" style="font-size:0.85rem">${h.description || '-'}</span></td>
            <td>${h.recurring ? '<span class="badge badge-success">Sí</span>' : '<span class="badge badge-secondary">No</span>'}</td>
            <td>
                <div class="actions">
                    <button class="btn btn-sm btn-secondary" onclick="editHoliday('${h.id}')" title="Editar">
                        <svg class="action-icon"><use href="#icon-edit"></use></svg>
                    </button>
                    <button class="btn btn-sm btn-secondary" onclick="deleteHoliday('${h.id}')" title="Eliminar">
                        <svg class="action-icon"><use href="#icon-trash"></use></svg>
                    </button>
                </div>
            </td>
        `;
        tbody.appendChild(tr);
    });

    rowCount.innerText = `Mostrando ${list.length} de ${list.length} registros`;
}

async function editHoliday(id) {
    try {
        const resp = await fetch(`/api/holidays/${id}`, { headers: getAuthHeaders() });
        const h = await resp.json();
        
        document.getElementById('holiday-modal-title').innerText = 'Editar Feriado';
        document.getElementById('holiday-id').value = h.id;
        document.getElementById('holiday-date').value = h.date.split('T')[0];
        document.getElementById('holiday-name').value = h.name;
        document.getElementById('holiday-description').value = h.description || '';
        document.getElementById('holiday-recurring').checked = h.recurring;
        
        document.getElementById('holiday-modal').classList.add('active');
    } catch (err) {
        showToast('Error al obtener feriado', 'error');
    }
}

async function deleteHoliday(id) {
    if (!confirm('¿Eliminar este feriado?')) return;
    try {
        const resp = await fetch(`/api/holidays/${id}`, {
            method: 'DELETE',
            headers: getAuthHeaders()
        });
        if (resp.ok) {
            showToast('Feriado eliminado');
            loadHolidays();
        }
    } catch (err) {
        showToast('Error al eliminar feriado', 'error');
    }
}

// Event Listeners for Holidays
document.addEventListener('DOMContentLoaded', () => {
    const btnAddHoliday = document.getElementById('btn-add-holiday');
    if (btnAddHoliday) {
        btnAddHoliday.addEventListener('click', () => {
            document.getElementById('holiday-modal-title').innerText = 'Nuevo Feriado';
            document.getElementById('holiday-form').reset();
            document.getElementById('holiday-id').value = '';
            document.getElementById('holiday-modal').classList.add('active');
        });
    }

    const holidayForm = document.getElementById('holiday-form');
    if (holidayForm) {
        holidayForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(holidayForm);
            const id = formData.get('id');
            const data = {
                name: formData.get('name'),
                date: new Date(formData.get('date')).toISOString(),
                description: formData.get('description'),
                recurring: holidayForm.querySelector('#holiday-recurring').checked
            };

            const url = id ? `/api/holidays/${id}` : '/api/holidays';
            const method = id ? 'PUT' : 'POST';

            try {
                const resp = await fetch(url, {
                    method: method,
                    headers: getAuthHeaders(),
                    body: JSON.stringify(data)
                });

                if (resp.ok) {
                    showToast(id ? 'Feriado actualizado' : 'Feriado creado');
                    document.getElementById('holiday-modal').classList.remove('active');
                    loadHolidays();
                } else {
                    const err = await resp.json();
                    showToast(err.error || 'Error al guardar feriado', 'error');
                }
            } catch (err) {
                showToast('Error de conexión', 'error');
            }
        });
    }
});

window.editHoliday = editHoliday;
window.deleteHoliday = deleteHoliday;
window.loadHolidays = loadHolidays;
