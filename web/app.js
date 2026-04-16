// Auth state
let currentUser = null;
let wsSocket = null;
let wsReconnectTimer = null;
let managedDevices = [];
let discoveredDevices = [];

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

document.addEventListener('DOMContentLoaded', () => {
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
    try {
        const resp = await fetch('/api/attendance/stats', {
            headers: getAuthHeaders()
        });
        if (!resp.ok) return;

        const stats = await resp.json();
        const mappings = {
            'stats-present': stats.present ?? 0,
            'stats-late': stats.late ?? 0,
            'stats-absent': stats.absent ?? 0
        };

        Object.entries(mappings).forEach(([id, value]) => {
            const el = document.getElementById(id);
            if (el) {
                el.innerText = value;
            }
        });

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
    }
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
        const resp = await fetch('/api/auth/me');

        if (resp.ok) {
            currentUser = await resp.json();
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

                hideLoginScreen();
                updateUserInfo();
                applyRolePermissions();
                initWebSocket();
                loadManagedDevices();
                loadDashboardStats();
                showToast('¡Bienvenido, ' + currentUser.fullName + '!');
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
    toggleRoleElement('btn-add-dept', canManageOrganization());
    toggleRoleElement('btn-add-pos', canManageOrganization());
    toggleRoleElement('btn-new-leave', canManageLeaves());
    toggleRoleElement('btn-new-travel', canManageTravel());
    toggleRoleElement('btn-manage-rates', canManageTravelRates());

    if (settingsNav) {
        settingsNav.classList.toggle('is-hidden-by-role', !canManageUsers());
    }
    if (settingsPage) {
        settingsPage.classList.toggle('is-hidden-by-role', !canManageUsers());
    }
    if (currentActivePage && ((!canManageUsers() && currentActivePage.id === 'settings') || (!canManageDevices() && currentActivePage.id === 'devices'))) {
        document.querySelector('.sidebar nav li[data-page="dashboard"]')?.click();
    }
}

function toggleRoleElement(id, visible) {
    const el = document.getElementById(id);
    if (!el) return;
    el.classList.toggle('is-hidden-by-role', !visible);
}

function getAuthHeaders() {
    return {
        'Content-Type': 'application/json'
    };
}

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

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const rawData = Object.fromEntries(formData.entries());
        const data = {
            company_name: rawData.company_name || '',
            company_rnc: rawData.company_rnc || '',
            grace_period_minutes: rawData.grace_period || '',
            overtime_threshold_hours: rawData.work_hours || ''
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
        };

        for (const [apiKey, fieldName] of Object.entries(fieldMap)) {
            const field = form.querySelector(`[name="${fieldName}"]`);
            if (field && config[apiKey] !== undefined) {
                field.value = config[apiKey];
            }
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
                const err = await resp.text();
                showToast(`Error: ${err}`, 'error');
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
                const err = await resp.text();
                showToast(`Error: ${err}`, 'error');
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

    closeBtns.forEach(btn => btn.addEventListener('click', () => modal.classList.remove('active')));

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const empNo = document.getElementById('face-emp-no').value;
        const formData = new FormData(form);

        const btn = form.querySelector('button[type="submit"]');
        btn.disabled = true;
        btn.innerText = 'Enviando...';

        try {
            const resp = await fetch(`/api/employees/${empNo}/face`, {
                method: 'POST',
                body: formData // Multipart
            });

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

        document.getElementById('employee-id').value = employee.id || '';
        form.querySelector('[name="firstName"]').value = employee.firstName || '';
        form.querySelector('[name="lastName"]').value = employee.lastName || '';
        form.querySelector('[name="idNumber"]').value = employee.idNumber || '';
        form.querySelector('[name="employeeNo"]').value = employee.employeeNo || '';
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
            document.getElementById('modal-title').innerText = 'Nuevo Empleado';
            if (submitBtn) {
                submitBtn.innerText = 'Guardar Empleado';
            }
            await loadEmployeeFormOptions();
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
        const employeeId = data.id;
        delete data.id;

        data.baseSalary = data.baseSalary ? parseFloat(data.baseSalary) : 0;
        data.status = data.status || 'Active';

        try {
            const url = employeeId ? `/api/employees/${employeeId}` : '/api/employees';
            const method = employeeId ? 'PUT' : 'POST';
            const resp = await fetch(url, {
                method: method,
                headers: getAuthHeaders(),
                body: JSON.stringify(data)
            });

            if (resp.ok) {
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
    document.getElementById('employee-id').value = '';
    const statusField = form.querySelector('[name="status"]');
    if (statusField) {
        statusField.value = 'Active';
    }
}

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
            if (pageId === 'attendance') loadAttendance();
            if (pageId === 'travel-allowances') loadTravelAllowances();
            if (pageId === 'leaves') loadLeaves();

            document.querySelector('.content').scrollTop = 0;
        });
    });

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
            if (pageId === 'devices') loadDevices();

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
    const wsUrl = `${protocol}//${window.location.host}/ws`;
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
                const resp = await fetch('/api/discovery/scan', {
                    method: 'POST',
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
            <td>${escapeHTML(dev.IPv4Address || '---')}</td>
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
        tbody.innerHTML = '<tr><td colspan="5" class="text-muted" style="text-align: center; padding: 2rem;">Aún no hay dispositivos administrados. Agrega uno manualmente o impórtalo desde SADP.</td></tr>';
        return;
    }

    tbody.innerHTML = managedDevices.map(device => `
        <tr>
            <td>
                <div class="device-identity">
                    <strong>${escapeHTML(device.name || '')}</strong>
                    <span class="text-muted">${escapeHTML(device.model || 'Terminal Hikvision')}</span>
                </div>
            </td>
            <td>${escapeHTML(device.ip || '')}:${device.port || 80}</td>
            <td>${escapeHTML(device.username || '---')} ${device.hasPassword ? '<span class="badge badge-success">OK</span>' : '<span class="badge badge-warning">Sin clave</span>'}</td>
            <td>${device.isDefault ? '<span class="badge badge-success">Predeterminado</span>' : '<span class="badge badge-secondary">Secundario</span>'}</td>
            <td>
                <div class="table-actions">
                    ${device.isDefault ? '' : `<button class="btn btn-sm btn-secondary" onclick="setManagedDeviceDefault(decodeInlineArg('${encodeInlineArg(device.id || '')}'))">Usar</button>`}
                    <div class="travel-actions">
                        <button class="btn-action btn-action--primary" onclick="editManagedDevice(decodeInlineArg('${encodeInlineArg(device.id || '')}'))">Editar</button>
                        <button class="btn-action btn-action--danger" onclick="deleteManagedDevice(decodeInlineArg('${encodeInlineArg(device.id || '')}'))">Eliminar</button>
                    </div>
                </div>
            </td>
        </tr>
    `).join('');
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

// Empleados
async function loadEmployees() {
    try {
        const resp = await fetch('/api/employees', {
            headers: getAuthHeaders()
        });
        const emps = await resp.json();
        const list = document.getElementById('employees-list');
        if (!list) return;

        if (!emps || emps.length === 0) {
            list.innerHTML = '<p class="text-muted" style="grid-column: 1/-1; text-align: center; padding: 3rem;">No hay empleados registrados. Haz clic en "+ Nuevo Empleado" para comenzar.</p>';
            return;
        }

        const canEditEmployees = canManageOrganization();
        const canManageFaces = canManageOrganization();
        list.innerHTML = emps.map(e => `
            <div class="employee-card">
                <div class="emp-avatar">👤</div>
                <div class="emp-info">
                    <h4>${escapeHTML(`${e.firstName || ''} ${e.lastName || ''}`.trim())}</h4>
                    <p class="text-muted">ID: ${escapeHTML(e.employeeNo || '')}</p>
                    <span class="badge ${e.status === 'Active' ? 'badge-success' : ''}">${escapeHTML(e.status || '')}</span>
                </div>
                <div class="emp-actions travel-actions">
                    ${canManageFaces ? `<button class="btn-action btn-action--view" onclick="openFaceModal(decodeInlineArg('${encodeInlineArg(e.employeeNo || '')}'))">Rostro</button>` : ''}
                    ${canEditEmployees ? `<button class="btn-action btn-action--primary" onclick="editEmployee(decodeInlineArg('${encodeInlineArg(e.id || '')}'))">Editar</button>` : ''}
                </div>
            </div>
        `).join('');
    } catch (err) {
        console.error('Load employees failed', err);
    }
}

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
                        <button class="btn-action btn-action--primary" onclick="editUser(decodeInlineArg('${encodeInlineArg(u.id || '')}'), decodeInlineArg('${encodeInlineArg(u.username || '')}'), decodeInlineArg('${encodeInlineArg(u.fullName || '')}'), decodeInlineArg('${encodeInlineArg(u.email || '')}'), decodeInlineArg('${encodeInlineArg(u.role || '')}'))">Editar</button>
                        ${u.username !== 'admin' ? `<button class="btn-action btn-action--danger" onclick="deleteUser(decodeInlineArg('${encodeInlineArg(u.id || '')}'))">Eliminar</button>` : ''}
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
            await loadEmployeesForDeptSelect();
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
        const resp = await fetch('/api/departments', {
            headers: getAuthHeaders()
        });
        const depts = await resp.json();
        const tbody = document.getElementById('departments-list');
        if (!tbody) return;

        if (!depts || depts.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" class="text-muted" style="text-align: center; padding: 2rem;">No hay departamentos registrados</td></tr>';
            return;
        }

        // Count employees per department
        const empResp = await fetch('/api/employees', { headers: getAuthHeaders() });
        const employees = await empResp.json();
        const empCount = {};
        employees.forEach(e => {
            if (e.departmentId) {
                empCount[e.departmentId] = (empCount[e.departmentId] || 0) + 1;
            }
        });

        const canEditDepartments = canManageOrganization();
        tbody.innerHTML = depts.map(d => `
            <tr>
                <td>
                    <strong>${escapeHTML(d.name || '')}</strong><br>
                    <small class="text-muted">Encargado: ${escapeHTML(d.managerName || 'Sin asignar')}</small>
                </td>
                <td>${escapeHTML(d.description || '---')}</td>
                <td>${empCount[d.id] || 0} empleados</td>
                <td class="travel-actions-cell">
                    <div class="travel-actions">
                        ${canEditDepartments ? `<button class="btn-action btn-action--primary" onclick="editDept(decodeInlineArg('${encodeInlineArg(d.id || '')}'), decodeInlineArg('${encodeInlineArg(d.name || '')}'), decodeInlineArg('${encodeInlineArg(d.description || '')}'), decodeInlineArg('${encodeInlineArg(d.managerId || '')}'))">Editar</button>` : '<span class="text-muted">Solo lectura</span>'}
                        ${canEditDepartments ? `<button class="btn-action btn-action--danger" onclick="deleteDept(decodeInlineArg('${encodeInlineArg(d.id || '')}'))">Eliminar</button>` : ''}
                    </div>
                </td>
            </tr>
        `).join('');
    } catch (err) {
        console.error('Load departments failed', err);
    }
}

async function loadEmployeesForDeptSelect() {
    try {
        const resp = await fetch('/api/employees', { headers: getAuthHeaders() });
        const emps = await resp.json();
        const select = document.getElementById('dept-manager');
        if (!select) return;
        select.innerHTML = '<option value="">Ninguno / Seleccionar después</option>' + 
            (emps || []).map(e => `<option value="${e.id}">${e.firstName} ${e.lastName}</option>`).join('');
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
        btnAdd.addEventListener('click', () => {
            document.getElementById('pos-modal-title').innerText = 'Nuevo Cargo';
            document.getElementById('pos-form').reset();
            document.getElementById('pos-id').value = '';
            document.getElementById('pos-level').value = '1';
            loadDepartmentsForSelect();
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
        const resp = await fetch('/api/positions', {
            headers: getAuthHeaders()
        });
        const positions = await resp.json();
        const tbody = document.getElementById('positions-list');
        if (!tbody) return;

        if (!positions || positions.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="text-muted" style="text-align: center; padding: 2rem;">No hay cargos registrados</td></tr>';
            return;
        }

        // Get departments names
        const deptResp = await fetch('/api/departments', { headers: getAuthHeaders() });
        const depts = await deptResp.json();
        const deptMap = {};
        depts.forEach(d => { deptMap[d.id] = d.name; });

        // Count employees per position
        const empResp = await fetch('/api/employees', { headers: getAuthHeaders() });
        const employees = await empResp.json();
        const empCount = {};
        employees.forEach(e => {
            if (e.positionId) {
                empCount[e.positionId] = (empCount[e.positionId] || 0) + 1;
            }
        });

        const canEditPositions = canManageOrganization();
        tbody.innerHTML = positions.map(p => `
            <tr>
                <td>${escapeHTML(p.name || '')}</td>
                <td>${escapeHTML(deptMap[p.departmentId] || '---')}</td>
                <td>${escapeHTML(p.level || '-')}</td>
                <td>${empCount[p.id] || 0} empleados</td>
                <td class="travel-actions-cell">
                    <div class="travel-actions">
                        ${canEditPositions ? `<button class="btn-action btn-action--primary" onclick="editPos(decodeInlineArg('${encodeInlineArg(p.id || '')}'), decodeInlineArg('${encodeInlineArg(p.name || '')}'), decodeInlineArg('${encodeInlineArg(p.departmentId || '')}'), decodeInlineArg('${encodeInlineArg(p.level || 1)}'))">Editar</button>` : '<span class="text-muted">Solo lectura</span>'}
                        ${canEditPositions ? `<button class="btn-action btn-action--danger" onclick="deletePos(decodeInlineArg('${encodeInlineArg(p.id || '')}'))">Eliminar</button>` : ''}
                    </div>
                </td>
            </tr>
        `).join('');
    } catch (err) {
        console.error('Load positions failed', err);
    }
}

window.editPos = (id, name, departmentId, level) => {
    document.getElementById('pos-modal-title').innerText = 'Editar Cargo';
    document.getElementById('pos-id').value = id;
    document.getElementById('pos-name').value = name;
    document.getElementById('pos-level').value = level;
    loadDepartmentsForSelect().then(() => {
        if (departmentId) {
            document.getElementById('pos-dept-select').value = departmentId;
        }
    });
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

// ==================== ATTENDANCE ====================

async function loadAttendance() {
    const dateInput = document.getElementById('attendance-date');
    if (!dateInput) return;

    const selectedDate = dateInput.value || new Date().toISOString().split('T')[0];
    dateInput.value = selectedDate;

    try {
        const resp = await fetch(`/api/attendance/daily?date=${selectedDate}`, {
            headers: getAuthHeaders()
        });
        const results = await resp.json();

        const tbody = document.querySelector('#attendance-table tbody');
        if (!tbody) return;

        if (!results || results.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="text-muted" style="text-align: center; padding: 2rem;">No hay registros de asistencia para esta fecha</td></tr>';
            return;
        }

        tbody.innerHTML = results.map(r => {
            const statusClass = r.isIncomplete ? 'badge-warning' : r.isAbsent ? 'badge-danger' : r.isLate ? 'badge-warning' : 'badge-success';
            const statusText = r.isIncomplete ? 'Incompleto' : r.isAbsent ? 'Ausente' : r.isLate ? 'Tarde' : 'Presente';
            const checkIn = r.checkIn ? r.checkIn.split('T')[1].substring(0, 5) : '---';
            const checkOut = r.checkOut ? r.checkOut.split('T')[1].substring(0, 5) : '---';
            const totalHours = typeof r.totalHours === 'number' ? r.totalHours.toFixed(2) : '0.00';
            const overtime = typeof r.overtime === 'number' ? r.overtime.toFixed(2) : '0.00';

            const notifyData = encodeInlineArg(JSON.stringify({
                employeeNo: r.employeeNo,
                employeeName: r.employeeName || r.employeeNo,
                date: selectedDate,
                status: statusText,
                checkIn: checkIn,
                checkOut: checkOut,
                department: r.department || ''
            }));

            return `
            <tr>
                <td>${escapeHTML(selectedDate)}</td>
                <td>${escapeHTML(r.employeeName || r.employeeNo)}</td>
                <td>${escapeHTML(checkIn)}</td>
                <td>${escapeHTML(checkOut)}</td>
                <td>${escapeHTML(totalHours)}</td>
                <td>${escapeHTML(overtime)}</td>
                <td><span class="badge ${statusClass}">${escapeHTML(statusText)}</span></td>
                <td>
                    <button class="btn btn-sm btn-secondary" onclick='openNotifyModal(JSON.parse(decodeInlineArg("${notifyData}")))' title="Reportar">
                        <svg viewBox="0 0 24 24" style="width:14px;height:14px"><use href="#icon-alert"></use></svg>
                    </button>
                </td>
            </tr>
            `;
        }).join('');
    } catch (err) {
        console.error('Load attendance failed', err);
    }
}

// Init attendance date filter
const attendanceDate = document.getElementById('attendance-date');
if (attendanceDate) {
    attendanceDate.value = new Date().toISOString().split('T')[0];
    attendanceDate.addEventListener('change', loadAttendance);
}

// ==================== VIATICOS ====================

let travelRatesCache = [];   // cached for calc preview
let travelEmpsCache = [];    // cached for calc preview
let travelCurrentFilter = 'all';
let travelDecisionId = null;
let travelDecisionType = null;

function getTravelSelectedEmployeeIds() {
    const requestType = document.getElementById('travel-request-type')?.value || 'single';
    if (requestType === 'group') {
        return Array.from(document.getElementById('travel-employees-group')?.selectedOptions || []).map(option => option.value).filter(Boolean);
    }
    const single = document.getElementById('travel-employee')?.value;
    return single ? [single] : [];
}

function setTravelRequestTypeUI() {
    const requestType = document.getElementById('travel-request-type')?.value || 'single';
    const singleWrap = document.getElementById('travel-employee')?.closest('.form-group');
    const groupWrap = document.getElementById('travel-group-wrap');
    const groupName = document.getElementById('travel-group-name');

    if (singleWrap) singleWrap.style.display = requestType === 'group' ? 'none' : '';
    if (groupWrap) groupWrap.style.display = requestType === 'group' ? '' : 'none';
    if (groupName) groupName.disabled = requestType !== 'group';
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

    // --- Filter bar ---
    document.querySelectorAll('.travel-filter-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.travel-filter-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
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
        const all = await resp.json();

        const filtered = travelCurrentFilter === 'all'
            ? all
            : all.filter(ta => ta.status === travelCurrentFilter);

        if (!filtered || filtered.length === 0) {
            tbody.innerHTML = `<tr><td colspan="9" style="text-align:center;padding:2rem;color:var(--text-muted);">Sin solicitudes${travelCurrentFilter === 'all' ? '' : ' con estado "' + formatTravelStatus(travelCurrentFilter) + '"'}.</td></tr>`;
            return;
        }

        tbody.innerHTML = filtered.map(ta => {
            const depDate = new Date(ta.departureDate).toLocaleDateString('es-ES', { day: '2-digit', month: 'short', year: 'numeric' });
            const retDate = new Date(ta.returnDate).toLocaleDateString('es-ES', { day: '2-digit', month: 'short', year: 'numeric' });
            const badge = travelStatusBadge(ta.status);
            const canEdit = ta.status === 'Pending' && canManageTravel();
            const canDecide = ta.status === 'Pending' && canManageTravel();
            const canDelete = ta.status === 'Pending' && canManageTravel();
            const summary = `${ta.employeeName || ''} - ${ta.destination || ''}`;
            const groupLabel = ta.groupSize > 1 ? (ta.groupName ? `${ta.groupName} (${ta.groupSize})` : `Solicitud grupal (${ta.groupSize})`) : '';

            return `
            <tr>
                <td style="font-weight:500;">
                    <div class="travel-meta-stack">
                        <span>${escapeHTML(ta.employeeName || '—')}</span>
                        ${groupLabel ? `<span class="travel-group-chip">${escapeHTML(groupLabel)}</span>` : ''}
                    </div>
                </td>
                <td>${escapeHTML(ta.destination || '')}</td>
                <td>${escapeHTML(depDate)}</td>
                <td>${escapeHTML(retDate)}</td>
                <td style="text-align:center;"><strong>${ta.days}</strong></td>
                <td>${ta.rateName ? `<span class="rate-type-badge rate-type-${escapeHTML(ta.rateType || '')}">${escapeHTML(ta.rateName)}</span>` : '—'}</td>
                <td style="font-weight:600;color:var(--accent);">RD$ ${Number(ta.calculatedAmount).toLocaleString('es-DO', { minimumFractionDigits: 2 })}</td>
                <td>${badge}</td>
                <td class="travel-actions-cell">
                    <div class="travel-actions">
                        <button class="btn-action btn-action--icon btn-action--view" onclick="downloadTravelAllowancePDF(decodeInlineArg('${encodeInlineArg(ta.id || '')}'))" title="Descargar PDF">📄</button>
                        ${canEdit ? `<button class="btn-action btn-action--primary" onclick="openTravelModal(decodeInlineArg('${encodeInlineArg(ta.id || '')}'))">Editar</button>` : ''}
                        ${canDecide ? `<button class="btn-action btn-action--icon btn-action--success" onclick="openTravelDecision(decodeInlineArg('${encodeInlineArg(ta.id || '')}'),'approve',decodeInlineArg('${encodeInlineArg(summary)}'))" title="Aprobar">✓</button>` : ''}
                        ${canDecide ? `<button class="btn-action btn-action--icon btn-action--danger" onclick="openTravelDecision(decodeInlineArg('${encodeInlineArg(ta.id || '')}'),'reject',decodeInlineArg('${encodeInlineArg(summary)}'))" title="Rechazar">✗</button>` : ''}
                        ${canDelete ? `<button class="btn-action btn-action--danger" onclick="deleteTravelAllowance(decodeInlineArg('${encodeInlineArg(ta.id || '')}'))">Eliminar</button>` : ''}
                    </div>
                </td>
            </tr>`;
        }).join('');
    } catch (err) {
        console.error('loadTravelAllowances failed', err);
        tbody.innerHTML = '<tr><td colspan="9" style="text-align:center;padding:2rem;color:var(--danger);">Error de conexión al cargar los viáticos.</td></tr>';
    }
}

async function openTravelModal(id = null) {
    // Load selects
    await Promise.all([loadTravelEmployeeSelect(), loadTravelRateSelect()]);

    const modal = document.getElementById('travel-modal');
    const form = document.getElementById('travel-form');
    form.reset();
    document.getElementById('travel-id').value = '';
    document.getElementById('travel-calc-preview').style.display = 'none';
    document.getElementById('travel-request-type').value = 'single';
    document.getElementById('travel-request-type').disabled = false;
    document.getElementById('travel-group-name').value = '';
    setTravelRequestTypeUI();

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
            if (ta.groupSize > 1) {
                const groupSelect = document.getElementById('travel-employees-group');
                Array.from(groupSelect.options).forEach(option => {
                    option.selected = option.value === ta.employeeId;
                });
                document.getElementById('travel-request-type').disabled = true;
            }
            setTravelRequestTypeUI();
            document.getElementById('travel-modal-title').innerText = 'Editar Solicitud de Viático';
            updateTravelCalcPreview();
        } catch (err) {
            showToast('Error de conexión', 'error'); return;
        }
    } else {
        document.getElementById('travel-modal-title').innerText = 'Nueva Solicitud de Viático';
    }

    modal.classList.add('active');
}

async function loadTravelEmployeeSelect() {
    try {
        const resp = await fetch('/api/employees', { headers: getAuthHeaders() });
        if (!resp.ok) return;
        const emps = await resp.json();
        travelEmpsCache = emps || [];
        const singleSelect = document.getElementById('travel-employee');
        const groupSelect = document.getElementById('travel-employees-group');
        if (!singleSelect || !groupSelect) return;

        const currentSingle = singleSelect.value;
        const currentGroup = new Set(Array.from(groupSelect.selectedOptions).map(option => option.value));
        const optionsHTML = travelEmpsCache.map(e => `<option value="${escapeHTML(e.id || '')}">${escapeHTML(`${e.firstName || ''} ${e.lastName || ''}`.trim())}</option>`).join('');

        singleSelect.innerHTML = '<option value="">Seleccionar empleado...</option>' + optionsHTML;
        singleSelect.value = currentSingle;

        groupSelect.innerHTML = optionsHTML;
        Array.from(groupSelect.options).forEach(option => {
            option.selected = currentGroup.has(option.value);
        });
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
    const employeeIds = getTravelSelectedEmployeeIds();
    const rateId = document.getElementById('travel-rate').value;
    const depVal = document.getElementById('travel-departure').value;
    const retVal = document.getElementById('travel-return').value;
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
                        <button class="btn-action btn-action--primary" onclick="editRate(decodeInlineArg('${encodeInlineArg(r.id || '')}'))">Editar</button>
                        <button class="btn-action btn-action--danger" onclick="deleteRate(decodeInlineArg('${encodeInlineArg(r.id || '')}'))">Eliminar</button>
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
    const applyBtn   = document.getElementById('ar-apply');
    const excelBtn   = document.getElementById('ar-export-excel');
    const pdfBtn     = document.getElementById('ar-export-pdf');
    const fromInput  = document.getElementById('ar-from');
    const toInput    = document.getElementById('ar-to');
    const deptSel    = document.getElementById('ar-dept');
    const statusSel  = document.getElementById('ar-status');
    const searchInp  = document.getElementById('ar-search');
    const loading    = document.getElementById('ar-loading');
    const emptyEl    = document.getElementById('ar-empty');
    const table      = document.getElementById('ar-table');
    const tbody      = document.getElementById('ar-tbody');
    const footer     = document.getElementById('ar-footer');
    const rowCount   = document.getElementById('ar-row-count');

    if (!applyBtn) return;

    // ── State ─────────────────────────────────────────────────────────────────
    let allRows     = [];   // All rows from last API call
    let sortCol     = 'date';
    let sortAsc     = true;

    // ── Defaults ──────────────────────────────────────────────────────────────
    const now = new Date();
    const firstDay = new Date(now.getFullYear(), now.getMonth(), 1);
    fromInput.value = firstDay.toISOString().split('T')[0];
    toInput.value   = now.toISOString().split('T')[0];

    window.loadAttendance = () => {
        loadDepts();
        fetchData();
    };

    // ── Department loader ─────────────────────────────────────────────────────
    async function loadDepts() {
        while (deptSel.options.length > 1) deptSel.remove(1);
        try {
            const resp = await fetch('/api/departments', { headers: getAuthHeaders() });
            if (!resp.ok) return;
            const depts = await resp.json();
            depts.forEach(d => {
                const opt = document.createElement('option');
                opt.value = d.id;
                opt.textContent = d.name;
                deptSel.appendChild(opt);
            });
        } catch (e) { /* silent */ }
    }

    // ── Data fetch ────────────────────────────────────────────────────────────
    async function fetchData() {
        const from = fromInput.value;
        const to   = toInput.value;
        if (!from || !to) { showToast('Selecciona el rango de fechas', 'error'); return; }
        if (from > to)    { showToast('Fecha inicio mayor a fecha fin', 'error'); return; }

        // Show loading
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
            renderTable();
        } catch (err) {
            showToast('Error al cargar datos: ' + err.message, 'error');
            loading.style.display = 'none';
        }
    }

    applyBtn.addEventListener('click', fetchData);
    fromInput.addEventListener('change', fetchData);
    toInput.addEventListener('change',   fetchData);
    deptSel.addEventListener('change',   fetchData);

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

    // ── Client-side filter + sort ─────────────────────────────────────────────
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

    // Live client-side search (no extra API call)
    searchInp.addEventListener('input', renderTable);
    statusSel.addEventListener('change', renderTable);

    function renderTable() {
        loading.style.display = 'none';
        const rows = getFiltered();

        if (rows.length === 0) {
            emptyEl.style.display = 'flex';
            table.style.display   = 'none';
            footer.style.display  = 'none';
            return;
        }

        // Sort
        const mult = sortAsc ? 1 : -1;
        rows.sort((a, b) => {
            const va = a[sortCol] ?? '';
            const vb = b[sortCol] ?? '';
            if (typeof va === 'number') return (va - vb) * mult;
            return va.localeCompare(vb) * mult;
        });

        emptyEl.style.display = 'none';
        table.style.display   = 'table';
        footer.style.display  = 'flex';

        tbody.innerHTML = rows.map(row => {
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

        rowCount.textContent = `${rows.length} registro${rows.length !== 1 ? 's' : ''}`;
        updateSortIcons();
    }

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

    // ── Export (respects current date+dept filters) ───────────────────────────
    function buildExportUrl(format) {
        const from = fromInput.value;
        const to   = toInput.value;
        const dept = deptSel.value;
        let url = `/api/reports/attendance?from=${from}&to=${to}&format=${format}`;
        if (dept) url += `&department=${encodeURIComponent(dept)}`;
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

    // --- Modal Closing ---
    modal.querySelectorAll('.close-modal').forEach(btn => {
        btn.addEventListener('click', () => modal.classList.remove('active'));
    });

    document.querySelectorAll('.leave-filter-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelectorAll('.leave-filter-btn').forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            const dataToFilter = document.getElementById('leaves-table').leavesData || [];
            renderLeaves(dataToFilter, e.target.dataset.status);
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

async function openNewLeaveModal() {
    const modal = document.getElementById('leave-modal');
    const form = document.getElementById('leave-form');
    if (!modal || !form) return;

    document.getElementById('leave-modal-title').innerText = 'Nuevo Permiso';
    form.reset();
    document.getElementById('leave-id').value = '';
    document.getElementById('leave-status').value = 'Approved';

    // Open first so the UI responds even if the background fetch is slow.
    modal.classList.add('active');

    await loadLeaveSelects();
}

async function loadLeaveSelects() {
    try {
        const [empResp, userResp] = await Promise.all([
            fetch('/api/employees', { headers: getAuthHeaders() }),
            fetch('/api/users', { headers: getAuthHeaders() })
        ]);
        const emps = await empResp.json();
        const users = await userResp.json();
        
        const empSel = document.getElementById('leave-employee');
        const authSel = document.getElementById('leave-authorized-by');
        
        if (empSel) {
            empSel.innerHTML = '<option value="">Seleccionar empleado...</option>' + 
                emps.map(e => `<option value="${escapeHTML(e.id || '')}">${escapeHTML(`${e.firstName || ''} ${e.lastName || ''}`.trim())}</option>`).join('');
        }
        if (authSel) {
            authSel.innerHTML = '<option value="">Seleccionar...</option>' + 
                users.map(u => `<option value="${escapeHTML(u.id || '')}">${escapeHTML(u.fullName || u.username || '')}</option>`).join('');
        }
    } catch(e) {}
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

    let filtered = leaves || [];
    if (filter !== 'all') {
        filtered = filtered.filter(l => l.status === filter);
    }

    if (filtered.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-muted text-center py-4">No hay permisos registrados</td></tr>';
        return;
    }

    const typeLabels = { 'Vacation': 'Vacaciones', 'Sick': 'Médico', 'Personal': 'Personal', 'Unpaid': 'Sin Goce', 'Other': 'Otro' };
    const statusBadges = {
        'Approved': '<span class="badge badge-success">Aprobado</span>',
        'Pending': '<span class="badge badge-warning">Pendiente</span>',
        'Rejected': '<span class="badge badge-danger">Rechazado</span>'
    };

    const canEditLeaves = canManageLeaves();
    tbody.innerHTML = filtered.map(l => `
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
                    ${canEditLeaves ? `<button class="btn-action btn-action--primary" onclick="editLeave(decodeInlineArg('${encodeInlineArg(l.id || '')}'))">Editar</button>` : '<span class="text-muted">Solo lectura</span>'}
                    ${canEditLeaves ? `<button class="btn-action btn-action--danger" onclick="deleteLeave(decodeInlineArg('${encodeInlineArg(l.id || '')}'))">Eliminar</button>` : ''}
                </div>
            </td>
        </tr>
    `).join('');
}

window.editLeave = async (id) => {
    try {
        const resp = await fetch(`/api/leaves/${id}`, { headers: getAuthHeaders() });
        const l = await resp.json();
        
        await loadLeaveSelects();
        document.getElementById('leave-modal-title').innerText = 'Editar Permiso';
        document.getElementById('leave-id').value = l.id;
        document.getElementById('leave-employee').value = l.employeeId;
        document.getElementById('leave-type').value = l.type;
        document.getElementById('leave-status').value = l.status;
        document.getElementById('leave-start').value = l.startDate ? l.startDate.split('T')[0] : '';
        document.getElementById('leave-end').value = l.endDate ? l.endDate.split('T')[0] : '';
        document.getElementById('leave-days').value = l.days;
        document.getElementById('leave-authorized-by').value = l.authorizedBy || '';
        document.getElementById('leave-reason').value = l.reason || '';
        document.getElementById('leave-notes').value = l.notes || '';
        document.getElementById('leave-modal').classList.add('active');
    } catch(e) {}
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
