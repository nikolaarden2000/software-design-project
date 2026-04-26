(function () {
  'use strict';

  let companies = [];
  let locations = [];
  let admins = [];
  let moderationRooms = [];

  document.addEventListener('DOMContentLoaded', initSuperuserPage);

  async function initSuperuserPage() {
    bindLogout();
    bindForms();

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
    fillLocationSelect();
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
    if (!root) return;

    if (companies.length === 0) {
      root.innerHTML = '<div class="item">Компаний пока нет</div>';
      return;
    }

    root.innerHTML = companies.map(company => `
      <div class="item">
        <div class="item-title">${escapeHtml(company.name)}</div>
        <div class="item-meta">${escapeHtml(company.description || 'Без описания')}</div>
        <div class="item-meta">Локаций: ${company.locations_count ?? 0}</div>
      </div>
    `).join('');
  }

  function renderLocations() {
    const root = document.getElementById('locationsList');
    if (!root) return;

    if (locations.length === 0) {
      root.innerHTML = '<div class="item">Локаций пока нет</div>';
      return;
    }

    root.innerHTML = locations.map(location => `
      <div class="item">
        <div class="item-title">${escapeHtml(location.company_name || 'Компания')}</div>
        <div class="item-meta">${escapeHtml(location.city)}, ${escapeHtml(location.address)}</div>
        <div class="item-meta">${escapeHtml(location.timezone || '')}</div>
      </div>
    `).join('');
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
        ? admin.locations.map(loc => `${loc.company_name}, ${loc.address}`).join('<br>')
        : 'Локации не назначены';

      return `
        <div class="item">
          <div class="item-title">${escapeHtml(admin.username)}</div>
          <div class="item-meta">${escapeHtml(admin.email)}</div>
          <div class="item-meta">${adminLocations}</div>
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

  function fillLocationSelect() {
    const select = document.getElementById('adminLocation');
    if (!select) return;

    select.innerHTML = '<option value="">Выберите локацию</option>';

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
        fillLocationSelect();
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

      const locationId = Number(document.getElementById('adminLocation').value);

      const payload = {
        username: document.getElementById('adminUsername').value.trim(),
        email: document.getElementById('adminEmail').value.trim(),
        password: document.getElementById('adminPassword').value
      };

      if (!payload.username || !payload.email || !payload.password || !locationId) {
        alert('Заполните данные администратора и выберите локацию');
        return;
      }

      try {
        const admin = await window.Api.createAdmin(payload);
        await window.Api.assignAdminToLocation(admin.id, locationId);

        form.reset();
        await loadAdmins();
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

  function escapeHtml(value) {
    return String(value ?? '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#039;');
  }
})();