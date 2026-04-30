
  'use strict';

  const ROOM_STATUSES = ['draft', 'pending', 'published', 'rejected', 'archived'];
  const BOOKING_STATUSES = ['booked', 'in_use', 'finished', 'canceled'];

  let locations = [];
  let currentRoom = null;

  document.addEventListener('DOMContentLoaded', initAdminPage);

  async function initAdminPage() {
    bindLogout();

    try {
      const me = await window.Api.getMe();

      if (!me?.authenticated) {
        window.location.href = '/auth';
        return;
      }

      if (me.user?.role !== 'admin' && me.user?.role !== 'superuser') {
        showAccessDenied();
        return;
      }

      showContent();
      await routeAdminPage();
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка загрузки admin-панели');
    }
  }

  function showAccessDenied() {
    document.getElementById('accessDenied')?.classList.remove('hidden');
    document.getElementById('adminContent')?.classList.add('hidden');
  }

  function showContent() {
    document.getElementById('accessDenied')?.classList.add('hidden');
    document.getElementById('adminContent')?.classList.remove('hidden');
  }

  async function routeAdminPage() {
    const path = window.location.pathname;

    if (path === '/admin') {
      await renderAdminHome();
      return;
    }

    const locationMatch = path.match(/^\/admin\/location\/(\d+)$/);
    if (locationMatch) {
      await renderLocationPage(Number(locationMatch[1]));
      return;
    }

    if (path === '/admin/room/new') {
      const params = new URLSearchParams(window.location.search);
      const locationId = Number(params.get('location_id'));

      if (!locationId) {
        renderError('Не указан location_id');
        return;
      }

      await renderNewRoomPage(locationId);
      return;
    }

    const roomMatch = path.match(/^\/admin\/room\/(\d+)$/);
    if (roomMatch) {
      await renderRoomPage(Number(roomMatch[1]));
      return;
    }

    renderError('Страница не найдена');
  }

  async function loadLocations() {
    const data = await window.Api.getAdminLocations();
    locations = data.items || [];
    return locations;
  }

  async function renderAdminHome() {
    const root = getRoot();

    root.innerHTML = `
      <div class="dashboard-header">
        <div>
          <h1>Мои локации</h1>
          <p>Выберите локацию, чтобы управлять помещениями.</p>
        </div>
      </div>

      <section class="panel panel-wide">
        <h2>Доступные локации</h2>
        <div id="locationsTable"></div>
      </section>
    `;

    await loadLocations();

    const tableRoot = document.getElementById('locationsTable');

    if (locations.length === 0) {
      tableRoot.innerHTML = '<div class="item">У вас пока нет назначенных локаций</div>';
      return;
    }

    tableRoot.innerHTML = `
      <table class="admin-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Компания</th>
            <th>Город</th>
            <th>Адрес</th>
            <th>Помещений</th>
            <th>Часовой пояс</th>
          </tr>
        </thead>
        <tbody>
          ${locations.map(location => `
            <tr class="clickable-row" data-location-id="${location.id}">
              <td>${location.id}</td>
              <td>${escapeHtml(location.company_name)}</td>
              <td>${escapeHtml(location.city)}</td>
              <td>${escapeHtml(location.address)}</td>
              <td>${location.rooms_count ?? 0}</td>
              <td>${escapeHtml(location.timezone || '')}</td>
            </tr>
          `).join('')}
        </tbody>
      </table>
    `;

    tableRoot.querySelectorAll('[data-location-id]').forEach(row => {
      row.addEventListener('click', () => {
        window.location.href = `/admin/location/${row.dataset.locationId}`;
      });
    });
  }

  async function renderLocationPage(locationId) {
    const root = getRoot();

    await loadLocations();

    const location = locations.find(item => Number(item.id) === Number(locationId));

    root.innerHTML = `
      <div class="dashboard-header">
        <div>
          <h1>${location ? escapeHtml(location.company_name) : 'Локация'}</h1>
          <p>${location ? `${escapeHtml(location.city)}, ${escapeHtml(location.address)}` : `ID локации: ${locationId}`}</p>
        </div>

        <div class="header-actions">
          <a class="btn" href="/admin">Назад к локациям</a>
          <a class="btn primary" href="/admin/room/new?location_id=${encodeURIComponent(locationId)}">
            Создать помещение
          </a>
        </div>
      </div>

      <section class="panel panel-wide">
        <div class="panel-header">
          <h2>Помещения локации</h2>

          <div class="filters-row">
            <select id="statusFilter">
              <option value="">Все статусы</option>
              ${ROOM_STATUSES.map(status => `
                <option value="${status}">${getRoomStatusLabel(status)}</option>
              `).join('')}
            </select>

            <button id="reloadRoomsBtn" class="btn">Обновить</button>
          </div>
        </div>

        <div id="roomsTable"></div>
      </section>
    `;

    document.getElementById('statusFilter')?.addEventListener('change', () => {
      loadRoomsForLocation(locationId);
    });

    document.getElementById('reloadRoomsBtn')?.addEventListener('click', () => {
      loadRoomsForLocation(locationId);
    });

    await loadRoomsForLocation(locationId);
  }

  async function loadRoomsForLocation(locationId) {
    const status = document.getElementById('statusFilter')?.value;

    const params = {
      location_id: locationId
    };

    if (status) {
      params.status = status;
    }

    const data = await window.Api.getAdminRooms(params);
    const rooms = data.items || [];

    renderRoomsTable(rooms);
  }

  function renderRoomsTable(rooms) {
    const tableRoot = document.getElementById('roomsTable');
    if (!tableRoot) return;

    if (rooms.length === 0) {
      tableRoot.innerHTML = '<div class="item">Помещений пока нет</div>';
      return;
    }

    tableRoot.innerHTML = `
      <table class="admin-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Название</th>
            <th>Вместимость</th>
            <th>Цена</th>
            <th>Статус</th>
            <th>Причина отклонения</th>
            <th>Дата создания</th>
          </tr>
        </thead>
        <tbody>
          ${rooms.map(room => `
            <tr class="clickable-row" data-room-id="${room.id}">
              <td>${room.id}</td>
              <td>${escapeHtml(room.title)}</td>
              <td>${room.capacity}</td>
              <td>${room.price} ₽/ч</td>
              <td>
                <span class="badge badge-${escapeHtml(room.status)}">
                  ${getRoomStatusLabel(room.status)}
                </span>
              </td>
              <td>${escapeHtml(room.rejection_reason || '—')}</td>
              <td>${formatDateTime(room.created_at)}</td>
            </tr>
          `).join('')}
        </tbody>
      </table>
    `;

    tableRoot.querySelectorAll('[data-room-id]').forEach(row => {
      row.addEventListener('click', () => {
        window.location.href = `/admin/room/${row.dataset.roomId}`;
      });
    });
  }

  async function renderNewRoomPage(locationId) {
    const root = getRoot();

    root.innerHTML = `
      <div class="dashboard-header">
        <div>
          <h1>Создание помещения</h1>
          <p>Локация ID: ${locationId}</p>
        </div>

        <div class="header-actions">
          <a class="btn" href="/admin/location/${encodeURIComponent(locationId)}">Назад</a>
        </div>
      </div>

      <section class="panel panel-wide">
        <h2>Новое помещение</h2>
        ${renderRoomForm({
          location_id: locationId,
          title: '',
          description: '',
          price: '',
          capacity: '',
          available_from: '09:00',
          available_to: '21:00',
          images: []
        }, false)}

        <div class="card-actions">
          <button id="saveDraftBtn" class="btn primary">Сохранить как черновик</button>
        </div>
      </section>
    `;

    document.getElementById('saveDraftBtn')?.addEventListener('click', async () => {
      const payload = readRoomForm();

      if (!validateRoomPayload(payload)) {
        return;
      }

      try {
        const created = await window.Api.createAdminRoom(payload);
        window.location.href = `/admin/room/${created.id}`;
      } catch (err) {
        console.error(err);
        alert(err?.message || 'Ошибка создания помещения');
      }
    });
  }

  async function renderRoomPage(roomId) {
    const root = getRoot();

    const data = await window.Api.getAdminRoom(roomId);
    currentRoom = data;

    const canEdit = currentRoom.status === 'draft' || currentRoom.status === 'rejected';
    const showBookings = ['published', 'pending', 'archived'].includes(currentRoom.status);

    root.innerHTML = `
      <div class="dashboard-header">
        <div>
          <h1>${escapeHtml(currentRoom.title)}</h1>
          <p>
            Статус:
            <span class="badge badge-${escapeHtml(currentRoom.status)}">
              ${getRoomStatusLabel(currentRoom.status)}
            </span>
          </p>
        </div>

        <div class="header-actions">
          <a class="btn" href="/admin/location/${encodeURIComponent(currentRoom.location_id)}">Назад к локации</a>
        </div>
      </div>

      <section class="panel panel-wide">
        <h2>Информация о помещении</h2>

        ${currentRoom.status === 'rejected' && currentRoom.rejection_reason ? `
          <div class="warning-box">
            Причина отклонения: ${escapeHtml(currentRoom.rejection_reason)}
          </div>
        ` : ''}

        ${renderRoomForm(currentRoom, !canEdit)}

        <div class="card-actions">
          ${canEdit ? `
            <button id="saveRoomBtn" class="btn primary">Сохранить изменения</button>
            <button id="submitRoomBtn" class="btn success">Отправить на модерацию</button>
          ` : ''}
        </div>
      </section>

      ${renderArchiveBlock(currentRoom)}

      ${showBookings ? `
        <section class="panel panel-wide">
          <div class="panel-header">
            <h2>Бронирования помещения</h2>

            <div class="filters-row">
              <select id="bookingStatusFilter">
                <option value="">Все статусы</option>
                ${BOOKING_STATUSES.map(status => `
                  <option value="${status}">${getBookingStatusLabel(status)}</option>
                `).join('')}
              </select>

              <button id="reloadBookingsBtn" class="btn">Обновить</button>
            </div>
          </div>

          <div id="bookingsTable"></div>
        </section>
      ` : ''}
    `;

    if (canEdit) {
      document.getElementById('saveRoomBtn')?.addEventListener('click', saveRoomChanges);
      document.getElementById('submitRoomBtn')?.addEventListener('click', submitCurrentRoom);
    }

    bindArchiveButtons();

    if (showBookings) {
      document.getElementById('bookingStatusFilter')?.addEventListener('change', () => {
        loadBookingsForRoom(roomId);
      });

      document.getElementById('reloadBookingsBtn')?.addEventListener('click', () => {
        loadBookingsForRoom(roomId);
      });

      await loadBookingsForRoom(roomId);
    }
  }

  function renderRoomForm(room, readonly) {
    const imagesValue = Array.isArray(room.images) ? room.images.join(', ') : '';

    return `
      <form id="roomForm" class="form">
        <input id="roomLocationId" type="hidden" value="${room.location_id || ''}" />

        <label>
          Название
          <input id="roomTitle" type="text" value="${escapeAttr(room.title)}" ${readonly ? 'readonly' : ''} required />
        </label>

        <label>
          Описание
          <textarea id="roomDescription" ${readonly ? 'readonly' : ''} required>${escapeHtml(room.description || '')}</textarea>
        </label>

        <div class="form-row">
          <label>
            Цена
            <input id="roomPrice" type="number" min="1" step="1" value="${room.price || ''}" ${readonly ? 'readonly' : ''} required />
          </label>

          <label>
            Вместимость
            <input id="roomCapacity" type="number" min="1" step="1" value="${room.capacity || ''}" ${readonly ? 'readonly' : ''} required />
          </label>
        </div>

        <div class="form-row">
          <label>
            Доступно с
            <input id="roomAvailableFrom" type="time" value="${room.available_from || '09:00'}" ${readonly ? 'disabled' : ''} required />
          </label>

          <label>
            Доступно до
            <input id="roomAvailableTo" type="time" value="${room.available_to || '21:00'}" ${readonly ? 'disabled' : ''} required />
          </label>
        </div>

        <label>
          Изображения
          <textarea id="roomImages" ${readonly ? 'readonly' : ''} placeholder="URL изображений через запятую">${escapeHtml(imagesValue)}</textarea>
        </label>
      </form>
    `;
  }

  function readRoomForm() {
    const imageValue = document.getElementById('roomImages').value.trim();

    return {
      location_id: Number(document.getElementById('roomLocationId').value),
      title: document.getElementById('roomTitle').value.trim(),
      description: document.getElementById('roomDescription').value.trim(),
      price: Number(document.getElementById('roomPrice').value),
      capacity: Number(document.getElementById('roomCapacity').value),
      available_from: document.getElementById('roomAvailableFrom').value,
      available_to: document.getElementById('roomAvailableTo').value,
      images: imageValue
        ? imageValue.split(',').map(item => item.trim()).filter(Boolean)
        : ['/shared/placeholders/room-placeholder.svg']
    };
  }

  function validateRoomPayload(payload) {
    if (!payload.location_id) {
      alert('Не указан location_id');
      return false;
    }

    if (!payload.title || !payload.description) {
      alert('Введите название и описание');
      return false;
    }

    if (payload.price <= 0 || payload.capacity <= 0) {
      alert('Цена и вместимость должны быть больше нуля');
      return false;
    }

    return true;
  }

  async function saveRoomChanges() {
    const payload = readRoomForm();

    if (!validateRoomPayload(payload)) {
      return;
    }

    try {
      await window.Api.updateAdminRoom(currentRoom.id, payload);
      alert('Изменения сохранены');
      await renderRoomPage(currentRoom.id);
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка сохранения помещения');
    }
  }

  async function submitCurrentRoom() {
    try {
      await window.Api.submitRoomForModeration(currentRoom.id);
      alert('Помещение отправлено на модерацию');
      await renderRoomPage(currentRoom.id);
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка отправки на модерацию');
    }
  }

  async function loadBookingsForRoom(roomId) {
    const status = document.getElementById('bookingStatusFilter')?.value;

    const params = {
      room_id: roomId
    };

    if (status) {
      params.status = status;
    }

    const data = await window.Api.getAdminBookings(params);
    const bookings = data.items || [];

    renderBookingsTable(bookings);
  }

  function renderBookingsTable(bookings) {
    const root = document.getElementById('bookingsTable');
    if (!root) return;

    if (bookings.length === 0) {
      root.innerHTML = '<div class="item">Бронирований пока нет</div>';
      return;
    }

    const sorted = [...bookings].sort((a, b) => {
      return `${a.date} ${a.start_time}`.localeCompare(`${b.date} ${b.start_time}`);
    });

    root.innerHTML = `
      <table class="admin-table">
        <thead>
          <tr>
            <th>ID брони</th>
            <th>Пользователь</th>
            <th>Email</th>
            <th>Дата</th>
            <th>Начало</th>
            <th>Окончание</th>
            <th>Стоимость</th>
            <th>Статус</th>
            <th>Действия</th>
          </tr>
        </thead>
        <tbody>
          ${sorted.map(booking => `
            <tr>
              <td>${booking.id}</td>
              <td>${escapeHtml(booking.user_username || '')}</td>
              <td>${escapeHtml(booking.user_email || '')}</td>
              <td>${escapeHtml(booking.date)}</td>
              <td>${escapeHtml(booking.start_time)}</td>
              <td>${escapeHtml(booking.end_time)}</td>
              <td>${booking.total_price} ₽</td>
              <td>${getBookingStatusLabel(booking.status)}</td>
              <td>
                ${booking.status === 'booked' ? `
                  <button class="btn danger small-btn" data-cancel-booking="${booking.id}">
                    Отменить
                  </button>
                ` : '—'}
              </td>
            </tr>
          `).join('')}
        </tbody>
      </table>
    `;

    root.querySelectorAll('[data-cancel-booking]').forEach(button => {
      button.addEventListener('click', () => cancelBooking(button.dataset.cancelBooking));
    });
  }

  async function cancelBooking(bookingId) {
    if (!confirm('Отменить бронирование?')) {
      return;
    }

    try {
      await window.Api.cancelAdminBooking(bookingId);
      await loadBookingsForRoom(currentRoom.id);
      alert('Бронирование отменено');
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка отмены бронирования');
    }
  }

  function renderArchiveBlock(room) {
    if (room.status === 'archived') {
      return `
        <section class="panel panel-wide">
          <h2>Архивирование</h2>
          <div class="info-box">Помещение уже архивировано.</div>
        </section>
      `;
    }

    const archive = room.archive || {};
    const bookingDisabled = archive.booking_disabled || room.booking_disabled;
    const scheduledFor = archive.scheduled_for || room.archive_scheduled_for;

    if (bookingDisabled || scheduledFor) {
      return `
        <section class="panel panel-wide">
          <h2>Архивирование</h2>
          <div class="warning-box">
            Помещение ожидает архивирования. Новые бронирования отключены.
            ${scheduledFor ? `<br>Запланировано на: ${formatDateTime(scheduledFor)}` : ''}
          </div>
        </section>
      `;
    }

    return `
      <section class="panel panel-wide">
        <h2>Архивирование</h2>

        ${archive.has_active_or_future_bookings ? `
          <div class="warning-box">
            У помещения есть активные или будущие бронирования.
            Немедленное архивирование может быть недоступно.
          </div>
        ` : ''}

        <div class="card-actions">
          <button id="archiveImmediateBtn" class="btn danger" ${archive.can_archive_now === false ? 'disabled' : ''}>
            Архивировать сейчас
          </button>

          <button id="archiveScheduledBtn" class="btn">
            Запланировать архивирование
          </button>
        </div>
      </section>
    `;
  }

  function bindArchiveButtons() {
    document.getElementById('archiveImmediateBtn')?.addEventListener('click', async () => {
      if (!confirm('Архивировать помещение сейчас?')) {
        return;
      }

      await archiveRoom('immediate');
    });

    document.getElementById('archiveScheduledBtn')?.addEventListener('click', async () => {
      if (!confirm('Отключить новые бронирования и запланировать архивирование после последней брони?')) {
        return;
      }

      await archiveRoom('scheduled');
    });
  }

  async function archiveRoom(mode) {
    try {
      await window.Api.archiveAdminRoom(currentRoom.id, mode);
      await renderRoomPage(currentRoom.id);
    } catch (err) {
      console.error(err);
      alert(err?.message || 'Ошибка архивирования помещения');
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

  function renderError(message) {
    getRoot().innerHTML = `
      <section class="access-denied">
        <h2>Ошибка</h2>
        <p>${escapeHtml(message)}</p>
        <a class="btn" href="/admin">В админ-панель</a>
      </section>
    `;
  }

  function getRoot() {
    return document.getElementById('adminRoot');
  }

  function getRoomStatusLabel(status) {
    const map = {
      draft: 'Черновик',
      pending: 'На модерации',
      published: 'Опубликовано',
      rejected: 'Отклонено',
      archived: 'Архивировано'
    };

    return map[status] || status;
  }

  function getBookingStatusLabel(status) {
    const map = {
      booked: 'Забронировано',
      in_use: 'Используется',
      finished: 'Завершено',
      canceled: 'Отменено'
    };

    return map[status] || status;
  }

  function formatDateTime(value) {
    if (!value) return '—';

    try {
      return new Date(value).toLocaleString('ru-RU');
    } catch {
      return value;
    }
  }

  function escapeHtml(value) {
    return String(value ?? '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#039;');
  }

  function escapeAttr(value) {
    return escapeHtml(value).replaceAll('`', '&#096;');
  }
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    initAdminPage,
    showAccessDenied,
    showContent,
    routeAdminPage,
    loadLocations,
    renderAdminHome,
    renderLocationPage,
    loadRoomsForLocation,
    renderRoomsTable,
    renderNewRoomPage,
    renderRoomPage,
    renderRoomForm,
    readRoomForm,
    validateRoomPayload,
    saveRoomChanges,
    submitCurrentRoom,
    loadBookingsForRoom,
    renderBookingsTable,
    cancelBooking,
    renderArchiveBlock,
    bindArchiveButtons,
    archiveRoom,
    bindLogout,
    renderError,
    getRoot,
    getRoomStatusLabel,
    getBookingStatusLabel,
    formatDateTime,
    escapeHtml,
    escapeAttr,
    __setCurrentRoomForTests: (room) => { currentRoom = room; },
__getCurrentRoomForTests: () => currentRoom
  };
}