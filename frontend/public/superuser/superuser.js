
  'use strict';

  let companies = [];
  let locations = [];
  let admins = [];
  let moderationRooms = [];
  let selectedAssignmentLocationIds = [];
  let activeLocationPickerRowIndex = null;

  const LIST_PREVIEW_LIMIT = 3;

  const expandedLists = {
    companies: false,
    locations: false,
    admins: false
  };

  document.addEventListener('DOMContentLoaded', initSuperuserPage);
  function renderLimitedList(root, items, type, emptyText, renderItem) {
  if (!root) return;

  if (!Array.isArray(items) || items.length === 0) {
    root.innerHTML = `<div class="item">${escapeHtml(emptyText)}</div>`;
    return;
  }

  const expanded = expandedLists[type];
  const visibleItems = expanded ? items : items.slice(0, LIST_PREVIEW_LIMIT);
  const hiddenCount = items.length - LIST_PREVIEW_LIMIT;

  root.innerHTML = `
    ${visibleItems.map(renderItem).join('')}

    ${items.length > LIST_PREVIEW_LIMIT ? `
      <button class="btn list-toggle-btn" data-list-toggle="${type}">
        ${expanded ? 'Свернуть' : `Показать все (${items.length})`}
      </button>
    ` : ''}
  `;

  root.querySelector('[data-list-toggle]')?.addEventListener('click', () => {
    expandedLists[type] = !expandedLists[type];

    if (type === 'companies') {
      renderCompanies();
    }

    if (type === 'locations') {
      renderLocations();
    }

    if (type === 'admins') {
      renderAdmins();
    }
  });
}

  async function initSuperuserPage() {
    bindLogout();
    bindForms();
    bindLocationPickerModal();

    try {
      const me = await window.Api.getMe();

      if (!me?.authenticated) {
        window.location.href = '/auth';
        return;
      }

      if (me.user?.role !== 'superuser') {
        showAccessDenied();
        return;
      }

      showContent();
      await loadAllData();
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка загрузки панели суперпользователя');
    }
  }

  function showAccessDenied() {
    document.getElementById('accessDenied')?.classList.remove('hidden');
    document.getElementById('superuserContent')?.classList.add('hidden');
  }

  function showContent() {
    document.getElementById('accessDenied')?.classList.add('hidden');
    document.getElementById('superuserContent')?.classList.remove('hidden');
  }

async function loadAllData() {
  await Promise.all([
    loadCompanies(),
    loadLocations(),
    loadAdmins(),
    loadModerationRooms()
  ]);

  fillCompanySelect();
  fillAssignmentSelects();
}

  async function loadCompanies() {
    const data = await window.Api.getCompanies();
    companies = data.items || [];
    renderCompanies();
  }

  async function loadLocations() {
    const data = await window.Api.getLocations();
    locations = data.items || [];
    renderLocations();
  }

  async function loadAdmins() {
    const data = await window.Api.getAdmins();
    admins = data.items || [];
    renderAdmins();
  }

  async function loadModerationRooms() {
    const data = await window.Api.getModerationRooms();
    moderationRooms = data.items || [];
    renderModerationRooms();
  }

function renderCompanies() {
  const root = document.getElementById('companiesList');

  renderLimitedList(
    root,
    companies,
    'companies',
    'Компаний пока нет',
    company => `
      <div class="item">
        <div class="item-title">${escapeHtml(company.name)}</div>
        <div class="item-meta">${escapeHtml(company.description || 'Без описания')}</div>
        <div class="item-meta">Локаций: ${company.locations_count ?? 0}</div>
      </div>
    `
  );
}

function renderLocations() {
  const root = document.getElementById('locationsList');

  renderLimitedList(
    root,
    locations,
    'locations',
    'Локаций пока нет',
    location => `
      <div class="item">
        <div class="item-title">${escapeHtml(location.company_name || 'Компания')}</div>
        <div class="item-meta">${escapeHtml(location.city)}, ${escapeHtml(location.address)}</div>
        <div class="item-meta">${escapeHtml(location.timezone || '')}</div>
      </div>
    `
  );
}
function renderAdmins() {
  const root = document.getElementById('adminsList');
  if (!root) return;

  if (admins.length === 0) {
    root.innerHTML = '<div class="item">Администраторов пока нет</div>';
    return;
  }

  root.innerHTML = admins.map(admin => {
    const adminLocations = Array.isArray(admin.locations) && admin.locations.length > 0
      ? admin.locations
        .map(loc => `${escapeHtml(loc.company_name)}, ${escapeHtml(loc.address)}`)
        .join('<br>')
      : 'Локации не назначены';

    return `
      <div class="item">
        <div class="item-title">${escapeHtml(admin.username)}</div>
        <div class="item-meta">${escapeHtml(admin.email)}</div>
        <div class="item-meta">
          <b>Локации:</b><br>
          ${adminLocations}
        </div>
      </div>
    `;
  }).join('');
}

  function renderModerationRooms() {
    const root = document.getElementById('moderationList');
    if (!root) return;

    if (moderationRooms.length === 0) {
      root.innerHTML = '<div class="item">Нет помещений на модерации</div>';
      return;
    }

    root.innerHTML = moderationRooms.map(room => `
      <article class="room-card" data-room-id="${room.id}">
        <h3>${escapeHtml(room.title)}</h3>

        <div class="item-meta">
          ${escapeHtml(room.company_name)} — ${escapeHtml(room.city)}, ${escapeHtml(room.address)}
        </div>

        <p>${escapeHtml(room.description || '')}</p>

        <div class="room-meta">
          <span class="badge">${room.price} ₽/ч</span>
          <span class="badge">до ${room.capacity} чел.</span>
          <span class="badge">${room.available_from} — ${room.available_to}</span>
          <span class="badge">${escapeHtml(room.status)}</span>
        </div>

        <div class="item-meta">
          Добавил: ${escapeHtml(room.created_by?.username || '')}
          (${escapeHtml(room.created_by?.email || '')})
        </div>

        <div class="reject-box">
          <textarea 
            id="rejectReason-${room.id}" 
            placeholder="Причина отклонения"
          ></textarea>
        </div>

        <div class="room-actions">
          <button class="btn success" data-action="approve" data-room-id="${room.id}">
            Одобрить
          </button>

          <button class="btn danger" data-action="reject" data-room-id="${room.id}">
            Отклонить
          </button>
        </div>
      </article>
    `).join('');

    root.querySelectorAll('[data-action="approve"]').forEach(button => {
      button.addEventListener('click', () => approveRoom(button.dataset.roomId));
    });

    root.querySelectorAll('[data-action="reject"]').forEach(button => {
      button.addEventListener('click', () => rejectRoom(button.dataset.roomId));
    });
  }

  function fillCompanySelect() {
    const select = document.getElementById('locationCompany');
    if (!select) return;

    select.innerHTML = '<option value="">Выберите компанию</option>';

    companies.forEach(company => {
      const option = document.createElement('option');
      option.value = company.id;
      option.textContent = company.name;
      select.appendChild(option);
    });
  }

    function fillAssignmentSelects() {
      fillAdminSelect();
      renderAssignmentLocationRows();
    }

    function fillAdminSelect() {
      const select = document.getElementById('assignAdmin');
      if (!select) return;

      const currentValue = select.value;

      select.innerHTML = '<option value="">Выберите администратора</option>';

      admins.forEach(admin => {
        const option = document.createElement('option');
        option.value = admin.id;
        option.textContent = `${admin.username} — ${admin.email}`;
        select.appendChild(option);
      });

      if (currentValue) {
        select.value = currentValue;
      }
    }

    function fillLocationsMultiSelect() {
      const select = document.getElementById('assignLocations');
      if (!select) return;

      select.innerHTML = '';

      locations.forEach(location => {
        const option = document.createElement('option');
        option.value = location.id;
        option.textContent = `${location.company_name} — ${location.city}, ${location.address}`;
        select.appendChild(option);
      });
    }

  function bindForms() {
    bindCompanyForm();
    bindLocationForm();
    bindAdminForm();
    bindAssignLocationsForm();
  }
  function bindCompanyForm() {
    const form = document.getElementById('companyForm');
    if (!form) return;

    form.addEventListener('submit', async (event) => {
      event.preventDefault();

      const payload = {
        name: document.getElementById('companyName').value.trim(),
        description: document.getElementById('companyDescription').value.trim()
      };

      if (!payload.name) {
        alert('Введите название компании');
        return;
      }

      try {
        await window.Api.createCompany(payload);
        form.reset();
        await loadCompanies();
        fillCompanySelect();
      } catch (err) {
        console.error(err);
        alert(err?.message || 'Ошибка создания компании');
      }
    });
  }

  function bindLocationForm() {
    const form = document.getElementById('locationForm');
    if (!form) return;

    form.addEventListener('submit', async (event) => {
      event.preventDefault();

      const payload = {
        company_id: Number(document.getElementById('locationCompany').value),
        city: document.getElementById('locationCity').value.trim(),
        address: document.getElementById('locationAddress').value.trim(),
        lat: Number(document.getElementById('locationLat').value),
        lng: Number(document.getElementById('locationLng').value),
        timezone: document.getElementById('locationTimezone').value.trim()
      };

      if (!payload.company_id || !payload.city || !payload.address) {
        alert('Заполните компанию, город и адрес');
        return;
      }

      try {
        await window.Api.createLocation(payload);
        form.reset();
        document.getElementById('locationTimezone').value = 'Europe/Moscow';

        await loadLocations();
        fillAssignmentSelects();
      } catch (err) {
        console.error(err);
        alert(err?.message || 'Ошибка создания локации');
      }
    });
  }

  function bindAdminForm() {
    const form = document.getElementById('adminForm');
    if (!form) return;

    form.addEventListener('submit', async (event) => {
      event.preventDefault();

      const payload = {
        username: document.getElementById('adminUsername').value.trim(),
        email: document.getElementById('adminEmail').value.trim(),
        password: document.getElementById('adminPassword').value
      };

      if (!payload.username || !payload.email || !payload.password) {
        alert('Заполните имя, email и пароль администратора');
        return;
      }

      try {
        await window.Api.createAdmin(payload);

        form.reset();

        await loadAdmins();
        fillAdminSelect();

        alert('Администратор создан. Теперь можно привязать к нему локации.');
      } catch (err) {
        console.error(err);
        alert(err?.message || 'Ошибка создания администратора');
      }
    });
  }

  async function approveRoom(roomId) {
    try {
      await window.Api.approveRoom(roomId);
      moderationRooms = moderationRooms.filter(room => String(room.id) !== String(roomId));
      renderModerationRooms();
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка одобрения помещения');
    }
  }

  async function rejectRoom(roomId) {
    const reasonInput = document.getElementById(`rejectReason-${roomId}`);
    const reason = reasonInput ? reasonInput.value.trim() : '';

    if (!reason) {
      alert('Введите причину отклонения');
      return;
    }

    try {
      await window.Api.rejectRoom(roomId, reason);
      moderationRooms = moderationRooms.filter(room => String(room.id) !== String(roomId));
      renderModerationRooms();
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка отклонения помещения');
    }
  }

  function bindLogout() {
    const logoutBtn = document.getElementById('logoutBtn');

    if (!logoutBtn) return;

    logoutBtn.addEventListener('click', async () => {
      try {
        await window.Api.logoutUser();
        window.location.href = '/';
      } catch (err) {
        console.error(err);
        alert(err?.message || 'Ошибка выхода');
      }
    });
  }
  function renderAssignmentLocationRows() {
  const root = document.getElementById('assignmentLocationRows');
  if (!root) return;

  const rows = [...selectedAssignmentLocationIds, null];

  root.innerHTML = rows.map((locationId, index) => {
    const location = locationId
      ? locations.find(item => Number(item.id) === Number(locationId))
      : null;

    if (!location) {
      return `
        <div class="location-picker-row location-picker-row-empty" data-location-row-index="${index}">
          <span>Выбрать локацию</span>
          <span class="location-picker-row__arrow">›</span>
        </div>
      `;
    }

    return `
      <div class="location-picker-row" data-location-row-index="${index}">
        <div>
          <div class="location-picker-row__title">
            ${escapeHtml(location.company_name || 'Компания')}
          </div>
          <div class="location-picker-row__meta">
            ${escapeHtml(location.city)}, ${escapeHtml(location.address)}
          </div>
        </div>

        <button
          type="button"
          class="location-picker-row__remove"
          data-remove-location-index="${index}"
        >
          Удалить
        </button>
      </div>
    `;
  }).join('');

  root.querySelectorAll('[data-location-row-index]').forEach(row => {
    row.addEventListener('click', (event) => {
      if (event.target.closest('[data-remove-location-index]')) {
        return;
      }

      openLocationPicker(Number(row.dataset.locationRowIndex));
    });
  });

  root.querySelectorAll('[data-remove-location-index]').forEach(button => {
    button.addEventListener('click', (event) => {
      event.stopPropagation();
      removeAssignmentLocation(Number(button.dataset.removeLocationIndex));
    });
  });
}

function removeAssignmentLocation(index) {
  selectedAssignmentLocationIds.splice(index, 1);
  renderAssignmentLocationRows();
}
function bindLocationPickerModal() {
  const searchInput = document.getElementById('locationPickerSearch');

  if (searchInput) {
    searchInput.addEventListener('input', () => {
      renderLocationPickerList(searchInput.value);
    });
  }

  document.querySelectorAll('[data-close-location-modal]').forEach(element => {
    element.addEventListener('click', closeLocationPicker);
  });

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      closeLocationPicker();
    }
  });
}

function openLocationPicker(rowIndex) {
  activeLocationPickerRowIndex = rowIndex;

  const modal = document.getElementById('locationPickerModal');
  const searchInput = document.getElementById('locationPickerSearch');

  if (searchInput) {
    searchInput.value = '';
  }

  renderLocationPickerList('');

  if (modal) {
    modal.classList.remove('hidden');
    modal.setAttribute('aria-hidden', 'false');
  }

  setTimeout(() => {
    searchInput?.focus();
  }, 0);
}

function closeLocationPicker() {
  activeLocationPickerRowIndex = null;

  const modal = document.getElementById('locationPickerModal');

  if (modal) {
    modal.classList.add('hidden');
    modal.setAttribute('aria-hidden', 'true');
  }
}

function renderLocationPickerList(query) {
  const root = document.getElementById('locationPickerList');
  if (!root) return;

  const normalizedQuery = String(query || '').trim().toLowerCase();

  const filtered = locations.filter(location => {
    const text = [
      location.company_name,
      location.city,
      location.address,
      location.timezone
    ].join(' ').toLowerCase();

    return text.includes(normalizedQuery);
  });

  if (filtered.length === 0) {
    root.innerHTML = '<div class="location-picker-empty">Локации не найдены</div>';
    return;
  }

  const currentLocationId = selectedAssignmentLocationIds[activeLocationPickerRowIndex];

  root.innerHTML = filtered.map(location => {
    const alreadySelected = selectedAssignmentLocationIds.some((id, index) => {
      return Number(id) === Number(location.id) && index !== activeLocationPickerRowIndex;
    });

    const isCurrent = Number(currentLocationId) === Number(location.id);

    return `
      <button
        type="button"
        class="location-option ${alreadySelected ? 'location-option-disabled' : ''}"
        data-location-id="${location.id}"
        ${alreadySelected ? 'disabled' : ''}
      >
        <div>
          <div class="location-option__title">
            ${escapeHtml(location.company_name || 'Компания')}
          </div>
          <div class="location-option__meta">
            ${escapeHtml(location.city)}, ${escapeHtml(location.address)}
          </div>
        </div>

        <div class="location-option__status">
          ${alreadySelected ? 'Уже выбрана' : isCurrent ? 'Выбрана' : 'Выбрать'}
        </div>
      </button>
    `;
  }).join('');

  root.querySelectorAll('[data-location-id]').forEach(button => {
    button.addEventListener('click', () => {
      selectLocationForAssignment(Number(button.dataset.locationId));
    });
  });
}

function selectLocationForAssignment(locationId) {
  if (activeLocationPickerRowIndex === null) {
    return;
  }

  const alreadySelectedInAnotherRow = selectedAssignmentLocationIds.some((id, index) => {
    return Number(id) === Number(locationId) && index !== activeLocationPickerRowIndex;
  });

  if (alreadySelectedInAnotherRow) {
    return;
  }

  if (activeLocationPickerRowIndex >= selectedAssignmentLocationIds.length) {
    selectedAssignmentLocationIds.push(locationId);
  } else {
    selectedAssignmentLocationIds[activeLocationPickerRowIndex] = locationId;
  }

  closeLocationPicker();
  renderAssignmentLocationRows();
}
function bindAssignLocationsForm() {
  const form = document.getElementById('assignLocationsForm');
  if (!form) return;

  form.addEventListener('submit', async (event) => {
    event.preventDefault();

    const adminId = Number(document.getElementById('assignAdmin').value);

    const uniqueLocationIds = [...new Set(
      selectedAssignmentLocationIds
        .map(Number)
        .filter(Boolean)
    )];

    if (!adminId) {
      alert('Выберите администратора');
      return;
    }

    if (uniqueLocationIds.length === 0) {
      alert('Выберите хотя бы одну локацию');
      return;
    }

    try {
      await Promise.all(
        uniqueLocationIds.map(locationId =>
          window.Api.assignAdminToLocation(adminId, locationId)
        )
      );

      selectedAssignmentLocationIds = [];

      form.reset();

      await loadAdmins();
      fillAdminSelect();
      renderAssignmentLocationRows();

      alert('Локации успешно привязаны к администратору');
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка привязки локаций');
    }
  });
}
  function escapeHtml(value) {
    return String(value ?? '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#039;');
  }
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    initSuperuserPage,

    showAccessDenied,
    showContent,

    loadAllData,
    loadCompanies,
    loadLocations,
    loadAdmins,
    loadModerationRooms,

    renderLimitedList,
    renderCompanies,
    renderLocations,
    renderAdmins,
    renderModerationRooms,

    fillCompanySelect,
    fillAssignmentSelects,
    fillAdminSelect,
    fillLocationsMultiSelect,

    bindForms,
    bindCompanyForm,
    bindLocationForm,
    bindAdminForm,
    bindAssignLocationsForm,

    approveRoom,
    rejectRoom,
    bindLogout,

    renderAssignmentLocationRows,
    removeAssignmentLocation,

    bindLocationPickerModal,
    openLocationPicker,
    closeLocationPicker,
    renderLocationPickerList,
    selectLocationForAssignment,

    escapeHtml,

    __setCompaniesForTests: (value) => { companies = value; },
    __getCompaniesForTests: () => companies,

    __setLocationsForTests: (value) => { locations = value; },
    __getLocationsForTests: () => locations,

    __setAdminsForTests: (value) => { admins = value; },
    __getAdminsForTests: () => admins,

    __setModerationRoomsForTests: (value) => { moderationRooms = value; },
    __getModerationRoomsForTests: () => moderationRooms,

    __setSelectedAssignmentLocationIdsForTests: (value) => { selectedAssignmentLocationIds = value; },
    __getSelectedAssignmentLocationIdsForTests: () => selectedAssignmentLocationIds,

    __setActiveLocationPickerRowIndexForTests: (value) => { activeLocationPickerRowIndex = value; },
    __getActiveLocationPickerRowIndexForTests: () => activeLocationPickerRowIndex,

    __resetStateForTests: () => {
      companies = [];
      locations = [];
      admins = [];
      moderationRooms = [];
      selectedAssignmentLocationIds = [];
      activeLocationPickerRowIndex = null;
      expandedLists.companies = false;
      expandedLists.locations = false;
      expandedLists.admins = false;
    }
  };
}