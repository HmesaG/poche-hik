(function () {
    const state = {
        employees: [],
        filtered: [],
    };

    const grid = document.getElementById("directory-grid");
    const searchInput = document.getElementById("directory-search");
    const departmentFilter = document.getElementById("department-filter");
    const statusFilter = document.getElementById("status-filter");
    const resultsSummary = document.getElementById("results-summary");
    const totalEmployees = document.getElementById("employees-total");
    const totalDepartments = document.getElementById("departments-total");
    const template = document.getElementById("employee-card-template");

    async function init() {
        bindEvents();
        await loadDirectory();
    }

    function bindEvents() {
        searchInput.addEventListener("input", applyFilters);
        departmentFilter.addEventListener("change", applyFilters);
        statusFilter.addEventListener("change", applyFilters);
    }

    async function loadDirectory() {
        try {
            const response = await fetch("/api/public/directory", {
                headers: {
                    Accept: "application/json",
                },
            });

            if (!response.ok) {
                throw new Error("No se pudo cargar el directorio");
            }

            const payload = await response.json();
            state.employees = Array.isArray(payload.employees) ? payload.employees : [];
            hydrateDepartmentFilter(state.employees);
            totalEmployees.textContent = String(state.employees.length);
            totalDepartments.textContent = String(
                new Set(state.employees.map((employee) => employee.departmentName).filter(Boolean)).size
            );
            applyFilters();
        } catch (error) {
            resultsSummary.textContent = error.message;
            grid.innerHTML = '<div class="empty-state">No fue posible cargar el directorio en este momento.</div>';
        }
    }

    function hydrateDepartmentFilter(employees) {
        const departments = [...new Set(employees.map((employee) => employee.departmentName).filter(Boolean))].sort();
        const options = ['<option value="">Todos</option>']
            .concat(departments.map((department) => `<option value="${escapeHtml(department)}">${escapeHtml(department)}</option>`));
        departmentFilter.innerHTML = options.join("");
    }

    function applyFilters() {
        const query = normalize(searchInput.value);
        const department = departmentFilter.value;
        const status = statusFilter.value;

        state.filtered = state.employees.filter((employee) => {
            const haystack = normalize([
                employee.fullName,
                employee.positionName,
                employee.departmentName,
                employee.phone,
                employee.email,
                employee.employeeNo,
            ].join(" "));

            const matchesQuery = !query || haystack.includes(query);
            const matchesDepartment = !department || employee.departmentName === department;
            const matchesStatus = !status || employee.status === status;

            return matchesQuery && matchesDepartment && matchesStatus;
        });

        render();
    }

    function render() {
        resultsSummary.textContent = `${state.filtered.length} contacto(s) visible(s) de ${state.employees.length}.`;

        if (!state.filtered.length) {
            grid.innerHTML = '<div class="empty-state">No hay empleados que coincidan con los filtros actuales.</div>';
            return;
        }

        const fragment = document.createDocumentFragment();
        state.filtered.forEach((employee) => {
            const node = template.content.firstElementChild.cloneNode(true);
            const anchorId = `empleado-${employee.employeeNo}`;
            node.id = anchorId;

            node.querySelector(".avatar").textContent = getInitials(employee.fullName);
            node.querySelector(".employee-name").textContent = employee.fullName || "Sin nombre";
            node.querySelector(".employee-role").textContent = employee.positionName || "Cargo no definido";
            node.querySelector(".employee-department").textContent = employee.departmentName || "Departamento no definido";

            const statusNode = node.querySelector(".employee-status");
            statusNode.textContent = formatStatus(employee.status);
            if (employee.status === "Active") {
                statusNode.classList.add("status-active");
            }

            fillContactValue(node.querySelector(".employee-phone"), employee.phone, "Sin telefono", "tel:");
            fillContactValue(node.querySelector(".employee-email"), employee.email, "Sin correo", "mailto:");
            node.querySelector(".employee-code").textContent = employee.employeeNo || "N/D";

            const vcfUrl = `/api/public/directory/${encodeURIComponent(employee.employeeNo)}/contact.vcf`;
            const profileUrl = `${window.location.origin}/directorio#${anchorId}`;
            const shareText = buildShareText(employee, profileUrl, vcfUrl);

            const vcfLink = node.querySelector(".action-vcf");
            vcfLink.href = vcfUrl;
            vcfLink.download = `${slugify(employee.fullName || employee.employeeNo || "contacto")}.vcf`;

            node.querySelector(".action-whatsapp").href = `https://wa.me/?text=${encodeURIComponent(shareText)}`;
            node.querySelector(".action-email").href = `mailto:?subject=${encodeURIComponent(`Contacto de ${employee.fullName}`)}&body=${encodeURIComponent(shareText)}`;
            node.querySelector(".action-share").addEventListener("click", () => shareEmployee(employee, shareText, profileUrl));

            if (window.location.hash === `#${anchorId}`) {
                node.classList.add("is-highlighted");
                setTimeout(() => node.scrollIntoView({ behavior: "smooth", block: "center" }), 50);
            }

            fragment.appendChild(node);
        });

        grid.innerHTML = "";
        grid.appendChild(fragment);
    }

    function buildShareText(employee, profileUrl, vcfUrl) {
        const lines = [
            `Contacto de ${employee.fullName}`,
            employee.positionName ? `Cargo: ${employee.positionName}` : "",
            employee.departmentName ? `Departamento: ${employee.departmentName}` : "",
            employee.phone ? `Telefono: ${employee.phone}` : "",
            employee.email ? `Correo: ${employee.email}` : "",
            `Directorio: ${profileUrl}`,
            `Descargar contacto: ${window.location.origin}${vcfUrl}`,
        ];

        return lines.filter(Boolean).join("\n");
    }

    async function shareEmployee(employee, shareText, profileUrl) {
        if (navigator.share) {
            try {
                await navigator.share({
                    title: employee.fullName,
                    text: shareText,
                    url: profileUrl,
                });
                return;
            } catch (error) {
                if (error && error.name === "AbortError") {
                    return;
                }
            }
        }

        if (navigator.clipboard && navigator.clipboard.writeText) {
            await navigator.clipboard.writeText(shareText);
            window.alert("La informacion del contacto fue copiada al portapapeles.");
            return;
        }

        window.prompt("Copia este contacto:", shareText);
    }

    function fillContactValue(node, value, fallback, hrefPrefix) {
        if (!value) {
            node.textContent = fallback;
            return;
        }

        const link = document.createElement("a");
        link.href = `${hrefPrefix}${value}`;
        link.textContent = value;
        node.appendChild(link);
    }

    function normalize(value) {
        return (value || "")
            .toString()
            .normalize("NFD")
            .replace(/[\u0300-\u036f]/g, "")
            .toLowerCase()
            .trim();
    }

    function getInitials(name) {
        return (name || "?")
            .split(" ")
            .filter(Boolean)
            .slice(0, 2)
            .map((part) => part.charAt(0).toUpperCase())
            .join("");
    }

    function formatStatus(status) {
        const labels = {
            Active: "Activo",
            Inactive: "Inactivo",
            Suspended: "Suspendido",
            Terminated: "Terminado",
        };
        return labels[status] || status || "Sin estado";
    }

    function slugify(value) {
        return normalize(value).replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "") || "contacto";
    }

    function escapeHtml(value) {
        return String(value)
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#39;");
    }

    init();
})();
